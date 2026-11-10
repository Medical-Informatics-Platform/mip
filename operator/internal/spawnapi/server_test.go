package spawnapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/time/rate"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	notebooksv1alpha1 "github.com/Medical-Informatics-Platform/mip/operator/api/v1alpha1"
	"github.com/Medical-Informatics-Platform/mip/operator/internal/controller"
)

const spawnAPIUser = "system:serviceaccount:mip-notebooks-system:mip-notebook-spawn-api"

// fakeAuth maps tokens to user names; "broken" stands for an API server that
// cannot be reached.
type fakeAuth map[string]string

func (f fakeAuth) Authenticate(_ context.Context, token string) (string, error) {
	if token == "broken" {
		return "", errors.New("connection refused")
	}
	if u, ok := f[token]; ok {
		return u, nil
	}
	return "", ErrUnauthenticated
}

// applyFile creates every object in a YAML file, tolerating ones that exist
// (cluster-scoped objects are shared by the specs).
func applyFile(path ...string) {
	data, err := os.ReadFile(filepath.Join(path...))
	Expect(err).NotTo(HaveOccurred())
	for _, doc := range strings.Split(string(data), "\n---\n") {
		obj := &unstructured.Unstructured{}
		Expect(utilyaml.Unmarshal([]byte(doc), &obj.Object)).To(Succeed())
		if err := k8sClient.Create(ctx, obj); !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

var _ = Describe("Spawn API", func() {
	var (
		ns      string
		srv     *Server
		counter int
	)

	path := func(name string) string { return "/v1/namespaces/" + ns + "/servers/" + name }
	do := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		switch b := body.(type) {
		case nil:
		case string:
			buf.WriteString(b)
		default:
			Expect(json.NewEncoder(&buf).Encode(b)).To(Succeed())
		}
		req := httptest.NewRequest(method, path, &buf)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	status := func(rec *httptest.ResponseRecorder) ServerStatus {
		out := ServerStatus{}
		Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
		return out
	}
	request := func() ServerRequest {
		return ServerRequest{
			Profile: "default", User: "alice", Env: map[string]string{"JUPYTERHUB_USER": "alice"},
			APIToken: "hub-api-token",
		}
	}
	fetch := func(name string) (*notebooksv1alpha1.Notebook, error) {
		nb := &notebooksv1alpha1.Notebook{}
		return nb, k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, nb)
	}
	newServer := func(c client.Client, limit rate.Limit, burst int) *Server {
		return NewServer(c, fakeAuth{
			"hub":           "system:serviceaccount:" + ns + ":jupyterhub",
			"other-hub":     "system:serviceaccount:federation-other:jupyterhub",
			"other-account": "system:serviceaccount:" + ns + ":default",
			"kube":          "system:serviceaccount:kube-system:jupyterhub",
		}, limit, burst, logr.Discard())
	}

	BeforeEach(func() {
		counter++
		ns = fmt.Sprintf("federation-%d", counter)
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})).To(Succeed())
		srv = newServer(k8sClient, rate.Inf, 0)
	})

	Context("authentication and authorization", func() {
		It("answers 401 without a bearer token", func() {
			rec := do(http.MethodGet, path("jupyter-alice"), "", nil)
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))
			Expect(rec.Body.String()).To(ContainSubstring(`"error"`))
		})
		It("answers 401 for a token the TokenReview rejects", func() {
			Expect(do(http.MethodGet, path("jupyter-alice"), "nope", nil).Code).To(Equal(http.StatusUnauthorized))
		})
		It("answers 503 when the token cannot be checked", func() {
			Expect(do(http.MethodGet, path("jupyter-alice"), "broken", nil).Code).To(Equal(http.StatusServiceUnavailable))
		})
		It("answers 403 to another federation's hub and creates nothing", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "other-hub", request()).Code).To(Equal(http.StatusForbidden))
			_, err := fetch("jupyter-alice")
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
		It("answers 403 to another account of the namespace", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "other-account", request()).Code).To(Equal(http.StatusForbidden))
		})
		It("answers 403 outside federation-* namespaces", func() {
			rec := do(http.MethodGet, "/v1/namespaces/kube-system/servers/jupyter-alice", "kube", nil)
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})
	})

	Context("PUT", func() {
		It("creates the Notebook, then its token Secret owned by it", func() {
			rec := do(http.MethodPut, path("jupyter-alice"), "hub", request())
			Expect(rec.Code).To(Equal(http.StatusCreated), rec.Body.String())

			nb, err := fetch("jupyter-alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(nb.Spec).To(Equal(notebooksv1alpha1.NotebookSpec{
				Profile: "default", User: "alice", Env: map[string]string{"JUPYTERHUB_USER": "alice"},
			}))
			Expect(nb.Annotations).To(HaveKeyWithValue(RequestedByAnnotation, "system:serviceaccount:"+ns+":jupyterhub"))

			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: "jupyter-alice-" + string(nb.UID) + "-token"}, secret)).To(Succeed())
			Expect(secret.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(secret.Data).To(Equal(map[string][]byte{"token": []byte("hub-api-token")}))
			Expect(secret.OwnerReferences).To(HaveLen(1))
			owner := secret.OwnerReferences[0]
			Expect(owner.Kind).To(Equal("Notebook"))
			Expect(owner.UID).To(Equal(nb.UID))
			Expect(*owner.Controller).To(BeTrue())
		})
		It("answers 200 to the same request again", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusCreated))
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusOK))
		})
		It("completes a Notebook whose Secret was never created", func() {
			nb := &notebooksv1alpha1.Notebook{
				ObjectMeta: metav1.ObjectMeta{Name: "jupyter-alice", Namespace: ns},
				Spec:       notebooksv1alpha1.NotebookSpec{Profile: "default", User: "alice", Env: map[string]string{"JUPYTERHUB_USER": "alice"}},
			}
			Expect(k8sClient.Create(ctx, nb)).To(Succeed())
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusOK))
			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: "jupyter-alice-" + string(nb.UID) + "-token"}, secret)).To(Succeed())
		})
		It("answers 409 when a Notebook with another spec exists", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusCreated))
			other := request()
			other.Profile = "other"
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", other).Code).To(Equal(http.StatusConflict))
			other = request()
			other.Env = nil
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", other).Code).To(Equal(http.StatusConflict))
		})
		It("answers 409 while the Notebook is terminating", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusCreated))
			// envtest has no garbage collector: a foreground delete stays terminating.
			Expect(do(http.MethodDelete, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusAccepted))
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusConflict))
		})
		It("rejects invalid requests before creating anything", func() {
			for name, want := range map[string]int{
				"alice":                              http.StatusBadRequest,
				"jupyter-Alice":                      http.StatusBadRequest,
				"jupyter-a.b":                        http.StatusBadRequest,
				"jupyter-" + strings.Repeat("a", 50): http.StatusBadRequest,
			} {
				Expect(do(http.MethodPut, path(name), "hub", request()).Code).To(Equal(want), name)
			}
			bad := func(edit func(r *ServerRequest)) ServerRequest {
				r := request()
				edit(&r)
				return r
			}
			for i, body := range []any{
				bad(func(r *ServerRequest) { r.Env["LD_PRELOAD"] = "/x.so" }),
				bad(func(r *ServerRequest) { r.Env["JUPYTERHUB_API_TOKEN"] = "x" }),
				bad(func(r *ServerRequest) { r.Env["JPY_API_TOKEN"] = "x" }),
				bad(func(r *ServerRequest) { r.APIToken = "" }),
				bad(func(r *ServerRequest) { r.Profile = "" }),
				`{"profile": "default", "user": "alice", "apiToken": "t", "image": "evil"}`,
				`{"profile": "default", "user": "alice", "apiToken": "t"} {}`,
				`not json`,
			} {
				Expect(do(http.MethodPut, path("jupyter-alice"), "hub", body).Code).To(Equal(http.StatusBadRequest), "body %d", i)
			}
			huge := bad(func(r *ServerRequest) { r.Env["JUPYTERHUB_X"] = strings.Repeat("x", 64<<10) })
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", huge).Code).To(Equal(http.StatusRequestEntityTooLarge))
			_, err := fetch("jupyter-alice")
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
		It("deletes the Notebook when its Secret cannot be created", func() {
			srv = newServer(interceptor.NewClient(k8sClient, interceptor.Funcs{
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*corev1.Secret); ok {
						return apierrors.NewInternalError(errors.New("etcd is down"))
					}
					return c.Create(ctx, obj, opts...)
				},
			}), rate.Inf, 0)
			rec := do(http.MethodPut, path("jupyter-alice"), "hub", request())
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).NotTo(ContainSubstring("etcd"))
			nb, err := fetch("jupyter-alice")
			if err == nil {
				Expect(nb.DeletionTimestamp).NotTo(BeNil(), "Notebook left without its Secret")
			} else {
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		})
	})

	Context("GET and DELETE", func() {
		It("reports the Notebook status", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusCreated))
			nb, err := fetch("jupyter-alice")
			Expect(err).NotTo(HaveOccurred())
			nb.Status = notebooksv1alpha1.NotebookStatus{Phase: "Running", URL: "http://10.0.0.7:8888", Message: "server ready"}
			Expect(k8sClient.Status().Update(ctx, nb)).To(Succeed())

			rec := do(http.MethodGet, path("jupyter-alice"), "hub", nil)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(status(rec)).To(Equal(ServerStatus{Phase: "Running", URL: "http://10.0.0.7:8888", Message: "server ready"}))
		})
		It("answers 404 for a server that does not exist", func() {
			Expect(do(http.MethodGet, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusNotFound))
			Expect(do(http.MethodDelete, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusNotFound))
		})
		It("answers 202 to DELETE until the Notebook is gone", func() {
			Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusCreated))
			Expect(do(http.MethodDelete, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusAccepted))

			rec := do(http.MethodGet, path("jupyter-alice"), "hub", nil)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(status(rec).Deleting).To(BeTrue())
			Expect(do(http.MethodDelete, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusAccepted))

			// What the garbage collector does once the pod is gone.
			nb, err := fetch("jupyter-alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(nb.Finalizers).To(ContainElement(metav1.FinalizerDeleteDependents))
			nb.Finalizers = nil
			Expect(k8sClient.Update(ctx, nb)).To(Succeed())
			Eventually(func() int {
				return do(http.MethodGet, path("jupyter-alice"), "hub", nil).Code
			}).Should(Equal(http.StatusNotFound))
			Expect(do(http.MethodDelete, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusNotFound))
		})
	})

	It("rate limits each namespace on its own", func() {
		srv = newServer(k8sClient, rate.Limit(0.1), 2)
		for range 2 {
			Expect(do(http.MethodGet, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusNotFound))
		}
		rec := do(http.MethodGet, path("jupyter-alice"), "hub", nil)
		Expect(rec.Code).To(Equal(http.StatusTooManyRequests))
		Expect(rec.Header().Get("Retry-After")).To(Equal("10"))

		rec = do(http.MethodGet, "/v1/namespaces/federation-other/servers/jupyter-alice", "other-hub", nil)
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})

	It("spawns as its own ServiceAccount, with only its RBAC, through the token Secret policy", func() {
		applyFile("..", "..", "testdata", "token-secrets-policy.yaml")
		applyFile("..", "..", "testdata", "spawn-api-role.yaml")
		Expect(k8sClient.Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "mip-notebook-spawn-api", Namespace: ns},
			Subjects: []rbacv1.Subject{{
				Kind: rbacv1.ServiceAccountKind, Name: "mip-notebook-spawn-api", Namespace: "mip-notebooks-system",
			}},
			RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "mip-notebook-spawn-api"},
		})).To(Succeed())
		spawnCfg := rest.CopyConfig(cfg)
		spawnCfg.Impersonate.UserName = spawnAPIUser
		spawnClient, err := client.New(spawnCfg, client.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())

		// The policy takes effect a moment after it is created.
		Eventually(func() error {
			return spawnClient.Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "keycloak-credentials", Namespace: ns},
				Type:       corev1.SecretTypeOpaque,
			})
		}).WithTimeout(10 * time.Second).Should(MatchError(ContainSubstring("mip-jupyterhub-secrets")))

		srv = newServer(spawnClient, rate.Inf, 0)
		Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusCreated))
		Expect(do(http.MethodPut, path("jupyter-alice"), "hub", request()).Code).To(Equal(http.StatusOK))
		Expect(do(http.MethodGet, path("jupyter-alice"), "hub", nil).Code).To(Equal(http.StatusOK))

		// Nothing beyond that.
		forbidden := func(err error) { ExpectWithOffset(1, apierrors.IsForbidden(err)).To(BeTrue(), "got %v", err) }
		forbidden(spawnClient.Create(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: ns},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "x", Image: "x"}}},
		}))
		forbidden(spawnClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: "keycloak-credentials"}, &corev1.Secret{}))
		forbidden(spawnClient.List(ctx, &notebooksv1alpha1.NotebookList{}, client.InNamespace(ns)))
		forbidden(spawnClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "jupyter-alice"}, &notebooksv1alpha1.Notebook{}))

		// The controller mounts the Secret the spawn API created.
		Expect(k8sClient.Create(ctx, &notebooksv1alpha1.NotebookProfile{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ns},
			Spec: notebooksv1alpha1.NotebookProfileSpec{
				Image:   "hbpmip/mip-jupyter:test",
				Storage: notebooksv1alpha1.NotebookStorage{Size: resource.MustParse("2Gi"), StorageClassName: "fast"},
			},
		})).To(Succeed())
		nb, err := fetch("jupyter-alice")
		Expect(err).NotTo(HaveOccurred())
		reconciler := &controller.NotebookReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(nb)})
		Expect(err).NotTo(HaveOccurred())
		pod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(nb), pod)).To(Succeed())
		var refs []string
		for _, e := range pod.Spec.Containers[0].Env {
			if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				refs = append(refs, e.ValueFrom.SecretKeyRef.Name+"/"+e.ValueFrom.SecretKeyRef.Key)
			}
		}
		Expect(refs).To(ConsistOf(
			"jupyter-alice-"+string(nb.UID)+"-token/token",
			"jupyter-alice-"+string(nb.UID)+"-token/token",
		))
		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: "jupyter-alice-" + string(nb.UID) + "-token"}, secret)).To(Succeed())
	})
})
