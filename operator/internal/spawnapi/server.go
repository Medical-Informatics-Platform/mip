// Package spawnapi is the HTTPS API the federation hubs use to start and stop
// notebook servers, so they hold no Kubernetes credential. It runs as its own
// process (--mode=spawn-api) and ServiceAccount, never inside the controller:
// the controller can create pods, so it must not listen where hubs reach it.
// See docs/notebook-operator.md.
package spawnapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notebooksv1alpha1 "github.com/Medical-Informatics-Platform/mip/operator/api/v1alpha1"
)

const (
	// RequestedByAnnotation keeps the hub account on the Notebook: the API
	// server only sees the spawn API's.
	RequestedByAnnotation = "notebooks.mip.ebrains.eu/requested-by"
	maxBodyBytes          = 64 << 10
	// Must match the pod the controller builds (internal/controller buildPod).
	tokenKey       = "token"
	maxNameLength  = 57
	cleanupTimeout = 10 * time.Second
)

var (
	federationNamespace = regexp.MustCompile(`^federation-[a-z0-9-]+$`)
	// Same rules as the CRD on Notebook.spec.env.
	allowedEnv = regexp.MustCompile(`^(JUPYTERHUB|JPY)_[A-Z0-9_]+$`)
	tokenEnv   = []string{"JUPYTERHUB_API_TOKEN", "JPY_API_TOKEN"}
)

// Server serves /v1/namespaces/{ns}/servers/{name}.
type Server struct {
	// Client must be uncached: the spawn API may not list or watch.
	Client  client.Client
	Auth    Authenticator
	Log     logr.Logger
	limiter *namespaceLimiter
	now     func() time.Time
}

func NewServer(c client.Client, auth Authenticator, limit rate.Limit, burst int, log logr.Logger) *Server {
	return &Server{Client: c, Auth: auth, Log: log, limiter: newNamespaceLimiter(limit, burst), now: time.Now}
}

// ServerRequest is the PUT body.
type ServerRequest struct {
	Profile    string            `json:"profile"`
	User       string            `json:"user"`
	ServerName string            `json:"serverName"`
	Env        map[string]string `json:"env"`
	APIToken   string            `json:"apiToken"`
}

// ServerStatus is what GET (and PUT) answer, from Notebook.status.
type ServerStatus struct {
	Phase    string `json:"phase"`
	URL      string `json:"url"`
	Message  string `json:"message"`
	Deleting bool   `json:"deleting"`
}

type handlerFunc func(w http.ResponseWriter, r *http.Request, ns, name, username string)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	const path = "/v1/namespaces/{ns}/servers/{name}"
	mux.Handle("PUT "+path, s.guard(s.put))
	mux.Handle("GET "+path, s.guard(s.get))
	mux.Handle("DELETE "+path, s.guard(s.delete))
	return instrument(mux)
}

// guard authenticates, authorizes and rate limits, in that order.
func (s *Server) guard(next handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("ns")
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			s.deny(w, r, ns, "", http.StatusUnauthorized, "missing_token", "bearer token required")
			return
		}
		username, err := s.Auth.Authenticate(r.Context(), token)
		if errors.Is(err, ErrUnauthenticated) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			s.deny(w, r, ns, "", http.StatusUnauthorized, "unauthenticated", "invalid token")
			return
		}
		if err != nil {
			s.Log.Error(err, "cannot authenticate request", "namespace", ns, "remoteAddr", r.RemoteAddr)
			writeError(w, http.StatusServiceUnavailable, "cannot verify the token; retry")
			return
		}
		if !federationNamespace.MatchString(ns) {
			s.deny(w, r, ns, username, http.StatusForbidden, "namespace", "not a federation namespace")
			return
		}
		// Each federation's hub manages its own namespace only.
		if username != "system:serviceaccount:"+ns+":jupyterhub" {
			s.deny(w, r, ns, username, http.StatusForbidden, "account", "account may not manage servers in this namespace")
			return
		}
		if ok, wait := s.limiter.allow(ns, s.now()); !ok {
			w.Header().Set("Retry-After", fmt.Sprint(int(math.Ceil(wait.Seconds()))))
			s.deny(w, r, ns, username, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		name := r.PathValue("name")
		if err := validName(name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		next(w, r, ns, name, username)
	})
}

func (s *Server) deny(w http.ResponseWriter, r *http.Request, ns, username string, code int, reason, msg string) {
	deniedTotal.WithLabelValues(reason).Inc()
	s.Log.Info("denied request", "namespace", ns, "remoteAddr", r.RemoteAddr, "reason", reason, "username", username)
	writeError(w, code, msg)
}

func (s *Server) put(w http.ResponseWriter, r *http.Request, ns, name, username string) {
	req, code, err := decodeRequest(w, r)
	if err != nil {
		writeError(w, code, err.Error())
		return
	}
	ctx := r.Context()
	log := s.Log.WithValues("namespace", ns, "notebook", name, "requestedBy", username)
	nb := &notebooksv1alpha1.Notebook{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: ns,
			Annotations: map[string]string{RequestedByAnnotation: username},
		},
		Spec: notebooksv1alpha1.NotebookSpec{
			Profile: req.Profile, User: req.User, ServerName: req.ServerName, Env: req.Env,
		},
	}
	err = s.Client.Create(ctx, nb)
	switch {
	case err == nil:
		if err := s.createSecret(ctx, nb, req.APIToken); err != nil {
			log.Error(err, "cannot create token Secret; deleting the Notebook")
			s.rollback(ctx, log, nb)
			writeError(w, http.StatusInternalServerError, "cannot create the token Secret")
			return
		}
		log.Info("created notebook", "uid", nb.UID)
		writeJSON(w, http.StatusCreated, statusOf(nb))
	case apierrors.IsAlreadyExists(err):
		s.retry(w, r, log, nb, req.APIToken)
	case apierrors.IsInvalid(err):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Error(err, "cannot create notebook")
		writeError(w, http.StatusInternalServerError, "cannot create the Notebook")
	}
}

// retry answers a PUT for a Notebook that exists: 200 when it is the same
// request again, 409 otherwise. The token is not compared: an existing Secret
// keeps the first request's. The hub deletes the old server before every PUT,
// so a same-spec PUT with a new token does not happen.
func (s *Server) retry(w http.ResponseWriter, r *http.Request, log logr.Logger, want *notebooksv1alpha1.Notebook, token string) {
	ctx := r.Context()
	nb := &notebooksv1alpha1.Notebook{}
	err := s.Client.Get(ctx, client.ObjectKeyFromObject(want), nb)
	switch {
	case apierrors.IsNotFound(err):
		writeError(w, http.StatusConflict, "the Notebook is being replaced; retry")
		return
	case err != nil:
		log.Error(err, "cannot get notebook")
		writeError(w, http.StatusInternalServerError, "cannot get the Notebook")
		return
	case nb.DeletionTimestamp != nil:
		writeError(w, http.StatusConflict, "the Notebook is terminating")
		return
	case !sameSpec(nb.Spec, want.Spec):
		writeError(w, http.StatusConflict, "a Notebook with a different spec exists")
		return
	}
	// No secrets get: creating it tells whether it exists, and completes a
	// request that stopped between the Notebook and its Secret.
	if err := s.createSecret(ctx, nb, token); err != nil {
		log.Error(err, "cannot create token Secret")
		writeError(w, http.StatusInternalServerError, "cannot create the token Secret")
		return
	}
	writeJSON(w, http.StatusOK, statusOf(nb))
}

// createSecret creates <name>-<uid>-token, owned by nb. One that exists can
// only be from an earlier request for this Notebook (the UID is in the name).
func (s *Server) createSecret(ctx context.Context, nb *notebooksv1alpha1.Notebook, token string) error {
	// Outlives a client that hangs up: a Notebook without its Secret is useless.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nb.Name + "-" + string(nb.UID) + "-token",
			Namespace: nb.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: notebooksv1alpha1.GroupVersion.String(),
				Kind:       "Notebook",
				Name:       nb.Name,
				UID:        nb.UID,
				Controller: ptr.To(true),
			}},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{tokenKey: []byte(token)},
	}
	if err := s.Client.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (s *Server) rollback(ctx context.Context, log logr.Logger, nb *notebooksv1alpha1.Notebook) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	// The UID precondition keeps a Notebook created since then.
	err := s.Client.Delete(ctx, nb, client.PropagationPolicy(metav1.DeletePropagationForeground),
		client.Preconditions{UID: &nb.UID})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "cannot delete notebook after failed start")
	}
}

func (s *Server) get(w http.ResponseWriter, r *http.Request, ns, name, _ string) {
	nb := &notebooksv1alpha1.Notebook{}
	err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, nb)
	switch {
	case apierrors.IsNotFound(err):
		writeError(w, http.StatusNotFound, "no such server")
	case err != nil:
		s.Log.Error(err, "cannot get notebook", "namespace", ns, "notebook", name)
		writeError(w, http.StatusInternalServerError, "cannot get the Notebook")
	default:
		writeJSON(w, http.StatusOK, statusOf(nb))
	}
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request, ns, name, username string) {
	nb := &notebooksv1alpha1.Notebook{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
	// Foreground: the Notebook goes after its pod, so the hub's wait for 404
	// also waits for the pod.
	err := s.Client.Delete(r.Context(), nb, client.PropagationPolicy(metav1.DeletePropagationForeground))
	switch {
	case apierrors.IsNotFound(err):
		writeError(w, http.StatusNotFound, "no such server")
	case err != nil:
		s.Log.Error(err, "cannot delete notebook", "namespace", ns, "notebook", name)
		writeError(w, http.StatusInternalServerError, "cannot delete the Notebook")
	default:
		s.Log.Info("deleting notebook", "namespace", ns, "notebook", name, "requestedBy", username)
		writeJSON(w, http.StatusAccepted, ServerStatus{Deleting: true})
	}
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (*ServerRequest, int, error) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	req := &ServerRequest{}
	err := dec.Decode(req)
	if err == nil && dec.Decode(&struct{}{}) != io.EOF {
		err = errors.New("trailing data after the JSON object")
	}
	if err != nil {
		if maxErr := (*http.MaxBytesError)(nil); errors.As(err, &maxErr) {
			return nil, http.StatusRequestEntityTooLarge, fmt.Errorf("body exceeds %d bytes", maxBodyBytes)
		}
		return nil, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err)
	}
	switch {
	case req.Profile == "":
		return nil, http.StatusBadRequest, errors.New("profile is required")
	case req.User == "":
		return nil, http.StatusBadRequest, errors.New("user is required")
	case req.APIToken == "":
		return nil, http.StatusBadRequest, errors.New("apiToken is required")
	}
	for k := range req.Env {
		if !allowedEnv.MatchString(k) {
			return nil, http.StatusBadRequest, fmt.Errorf("env key %q must match %s", k, allowedEnv)
		}
		for _, t := range tokenEnv {
			if k == t {
				return nil, http.StatusBadRequest, fmt.Errorf("env may not set %s; send apiToken", k)
			}
		}
	}
	return req, 0, nil
}

// validName applies the CRD's name rule; hub names are DNS labels.
func validName(name string) error {
	if !strings.HasPrefix(name, "jupyter-") || len(name) > maxNameLength {
		return fmt.Errorf("name must start with jupyter- and be at most %d characters", maxNameLength)
	}
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		return fmt.Errorf("name: %s", strings.Join(errs, "; "))
	}
	return nil
}

func sameSpec(a, b notebooksv1alpha1.NotebookSpec) bool {
	if a.Profile != b.Profile || a.User != b.User || a.ServerName != b.ServerName || len(a.Env) != len(b.Env) {
		return false
	}
	for k, v := range a.Env {
		if w, ok := b.Env[k]; !ok || w != v {
			return false
		}
	}
	return true
}

func statusOf(nb *notebooksv1alpha1.Notebook) ServerStatus {
	return ServerStatus{
		Phase:    nb.Status.Phase,
		URL:      nb.Status.URL,
		Message:  nb.Status.Message,
		Deleting: nb.DeletionTimestamp != nil,
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
