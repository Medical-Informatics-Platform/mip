package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	notebooksv1alpha1 "github.com/Medical-Informatics-Platform/mip/operator/api/v1alpha1"
)

const (
	notebookPort     = 8888
	homeVolume       = "home"
	profileRetryWait = 30 * time.Second
)

// NotebookReconciler turns Notebooks into a home PVC and a notebook pod.
type NotebookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// ExamplesClaim, when set and bound in the notebook's namespace, is mounted read-only
	// at work/.mip-examples in the home of every new notebook.
	ExamplesClaim string
}

// +kubebuilder:rbac:groups=notebooks.mip.ebrains.eu,resources=notebooks,verbs=get;list;watch
// +kubebuilder:rbac:groups=notebooks.mip.ebrains.eu,resources=notebooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=notebooks.mip.ebrains.eu,resources=notebookprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods;persistentvolumeclaims,verbs=get;list;watch;create

func (r *NotebookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	nb := &notebooksv1alpha1.Notebook{}
	if err := r.Get(ctx, req.NamespacedName, nb); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Pod and token Secret are owned by the Notebook; garbage collection
	// removes them, so deletion needs no finalizer.
	if !nb.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	pod := &corev1.Pod{}
	err := r.Get(ctx, client.ObjectKeyFromObject(nb), pod)
	if apierrors.IsNotFound(err) {
		return r.start(ctx, nb)
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	// Same name, other owner: a KubeSpawner pod, or an earlier Notebook's. Its IP
	// would send the hub to a server with another API token.
	if !metav1.IsControlledBy(pod, nb) {
		msg := fmt.Sprintf("pod %s exists and does not belong to this Notebook", pod.Name)
		return ctrl.Result{}, r.setStatus(ctx, nb, notebooksv1alpha1.PhaseFailed, "", msg)
	}

	phase, url, msg := podStatus(pod)
	return ctrl.Result{}, r.setStatus(ctx, nb, phase, url, msg)
}

// start creates the home and the pod. Only here is the profile needed: a running
// pod does not depend on it, and "Failed" makes the hub stop the server.
func (r *NotebookReconciler) start(ctx context.Context, nb *notebooksv1alpha1.Notebook) (ctrl.Result, error) {
	profile := &notebooksv1alpha1.NotebookProfile{}
	err := r.Get(ctx, types.NamespacedName{Namespace: nb.Namespace, Name: nb.Spec.Profile}, profile)
	if apierrors.IsNotFound(err) {
		msg := fmt.Sprintf("NotebookProfile %q not found", nb.Spec.Profile)
		return ctrl.Result{RequeueAfter: profileRetryWait}, r.setStatus(ctx, nb, notebooksv1alpha1.PhaseFailed, "", msg)
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.createPod(ctx, nb, profile); err != nil {
		// Pending, not Failed: a quota can free up. The message reaches the spawn
		// page, which otherwise shows nothing until the hub's timeout.
		return ctrl.Result{}, errors.Join(err, r.setStatus(ctx, nb, notebooksv1alpha1.PhasePending, "", "cannot create pod: "+err.Error()))
	}
	return ctrl.Result{}, r.setStatus(ctx, nb, notebooksv1alpha1.PhasePending, "", "pod created")
}

func (r *NotebookReconciler) createPod(ctx context.Context, nb *notebooksv1alpha1.Notebook, profile *notebooksv1alpha1.NotebookProfile) error {
	if err := r.ensurePVC(ctx, nb, profile); err != nil {
		return err
	}
	pod := buildPod(nb, profile)
	bound, err := r.examplesBound(ctx, nb.Namespace)
	if err != nil {
		return err
	}
	examples := "not bound"
	switch {
	case bound && !profile.Spec.Storage.FixOwnership:
		// Without fix-home-perm the runtime creates work/ root-owned for the nested mount.
		examples = "skipped: fixOwnership is false"
	case bound:
		addExamples(pod, r.ExamplesClaim, profile.Spec.Storage.MountPath)
		examples = "mounted"
	}
	if err := controllerutil.SetControllerReference(nb, pod, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	logf.FromContext(ctx).Info("created notebook pod", "pod", pod.Name, "examplesClaim", r.ExamplesClaim, "examples", examples)
	return nil
}

// ensurePVC creates the user's home volume. It is never owned, updated or
// deleted by the operator, so homes survive server restarts.
func (r *NotebookReconciler) ensurePVC(ctx context.Context, nb *notebooksv1alpha1.Notebook, profile *notebooksv1alpha1.NotebookProfile) error {
	pvc := &corev1.PersistentVolumeClaim{}
	key := types.NamespacedName{Namespace: nb.Namespace, Name: pvcName(nb.Spec.User)}
	err := r.Get(ctx, key, pvc)
	if !apierrors.IsNotFound(err) {
		return err
	}
	pvc = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels:    notebookLabels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: profile.Spec.Storage.Size},
			},
		},
	}
	if sc := profile.Spec.Storage.StorageClassName; sc != "" {
		pvc.Spec.StorageClassName = &sc
	}
	if err := r.Create(ctx, pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func buildPod(nb *notebooksv1alpha1.Notebook, profile *notebooksv1alpha1.NotebookProfile) *corev1.Pod {
	ps := profile.Spec

	// The entrypoint's work dir, on the home volume: fix-home-perm creates it and the
	// examples mount sits in it. First, so profile or notebook env can still override it.
	env := append([]corev1.EnvVar{{Name: "MIP_WORK_DIR", Value: path.Join(ps.Storage.MountPath, "work")}}, ps.Env...)
	for _, k := range slices.Sorted(maps.Keys(nb.Spec.Env)) {
		env = append(env, corev1.EnvVar{Name: k, Value: nb.Spec.Env[k]})
	}
	for _, name := range []string{"JUPYTERHUB_API_TOKEN", "JPY_API_TOKEN"} {
		env = append(env, corev1.EnvVar{Name: name, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				// Never read by the operator: until the hub has created it the
				// pod waits in CreateContainerConfigError, shown as the message.
				// The UID keeps out a Secret left by an earlier Notebook of the
				// same name.
				LocalObjectReference: corev1.LocalObjectReference{Name: nb.TokenSecretName()},
				Key:                  notebooksv1alpha1.TokenSecretKey,
			},
		}})
	}

	restricted := &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nb.Name,
			Namespace: nb.Namespace,
			Labels:    notebookLabels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                corev1.RestartPolicyNever,
			AutomountServiceAccountToken: ptr.To(false),
			EnableServiceLinks:           ptr.To(false),
			NodeSelector:                 ps.NodeSelector,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:           ptr.To[int64](1000),
				RunAsGroup:          ptr.To[int64](100),
				FSGroup:             ptr.To[int64](100),
				FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
				RunAsNonRoot:        ptr.To(true),
				SeccompProfile:      &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{{
				Name:            "notebook",
				Image:           ps.Image,
				ImagePullPolicy: ps.ImagePullPolicy,
				Resources:       ps.Resources,
				Env:             env,
				Ports:           []corev1.ContainerPort{{Name: "notebook", ContainerPort: notebookPort}},
				VolumeMounts:    []corev1.VolumeMount{{Name: homeVolume, MountPath: ps.Storage.MountPath}},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler:  corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(notebookPort)}},
					PeriodSeconds: 2,
				},
				SecurityContext: restricted,
			}},
			Volumes: []corev1.Volume{{
				Name: homeVolume,
				VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName(nb.Spec.User),
				}},
			}},
		},
	}
	if ps.Storage.FixOwnership {
		// Runs as root, so a profile using it cannot run under PSS "restricted".
		pod.Spec.InitContainers = []corev1.Container{{
			Name:  "fix-home-perm",
			Image: ps.Image,
			Command: []string{"sh", "-c",
				`[ "$(stat -c %u:%g /mnt/home)" = 1000:100 ] || chown -R 1000:100 /mnt/home; ` +
					"mkdir -p /mnt/home/work && chown 1000:100 /mnt/home/work"},
			VolumeMounts: []corev1.VolumeMount{{Name: homeVolume, MountPath: "/mnt/home"}},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:    ptr.To[int64](0),
				RunAsGroup:   ptr.To[int64](0),
				RunAsNonRoot: ptr.To(false),
			},
		}}
	}
	return pod
}

// examplesBound reports whether ns holds the shared examples claim, bound and not being
// deleted. A missing or unbound claim is false, not an error.
func (r *NotebookReconciler) examplesBound(ctx context.Context, ns string) (bool, error) {
	if r.ExamplesClaim == "" {
		return false, nil
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: r.ExamplesClaim}, pvc); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return pvc.DeletionTimestamp.IsZero() && pvc.Status.Phase == corev1.ClaimBound, nil
}

// addExamples mounts the claim at work/.mip-examples inside the home volume, where the
// single-user entrypoint links examples/ to it. The runtime creates the mount point, as
// root, in the parent directory, so work/ must already exist: fix-home-perm creates it.
func addExamples(pod *corev1.Pod, claim, home string) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "examples",
		VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: claim,
			ReadOnly:  true,
		}},
	})
	c := &pod.Spec.Containers[0]
	// A path in the Linux container, not on the operator's OS, so never filepath.Join.
	// nosemgrep: semgrep-rules.go.lang.correctness.use-filepath-join
	c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{Name: "examples", MountPath: path.Join(home, "work/.mip-examples"), ReadOnly: true})
}

// podStatus maps a pod to the Notebook phase, proxy URL and spawn message.
func podStatus(pod *corev1.Pod) (string, string, string) {
	switch pod.Status.Phase {
	case corev1.PodSucceeded, corev1.PodFailed:
		return notebooksv1alpha1.PhaseFailed, "", terminationMessage(pod)
	}
	if !pod.DeletionTimestamp.IsZero() {
		return notebooksv1alpha1.PhaseFailed, "", "pod is being deleted"
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue && pod.Status.PodIP != "" {
			return notebooksv1alpha1.PhaseRunning, fmt.Sprintf("http://%s:%d", pod.Status.PodIP, notebookPort), "server ready"
		}
	}
	return notebooksv1alpha1.PhasePending, "", waitingMessage(pod)
}

func waitingMessage(pod *corev1.Pod) string {
	statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
	for _, cs := range statuses {
		if w := cs.State.Waiting; w != nil && w.Reason != "" {
			return strings.TrimSpace(w.Reason + ": " + w.Message)
		}
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse {
			return strings.TrimSpace(c.Reason + ": " + c.Message)
		}
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running != nil {
			return "starting notebook server"
		}
	}
	return "pod " + strings.ToLower(string(pod.Status.Phase))
}

func terminationMessage(pod *corev1.Pod) string {
	for _, cs := range pod.Status.InitContainerStatuses {
		if t := cs.State.Terminated; t != nil && t.ExitCode != 0 {
			return strings.TrimSpace(fmt.Sprintf("init container %s exited with code %d: %s %s", cs.Name, t.ExitCode, t.Reason, t.Message))
		}
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if t := cs.State.Terminated; t != nil {
			return strings.TrimSpace(fmt.Sprintf("notebook exited with code %d: %s %s", t.ExitCode, t.Reason, t.Message))
		}
	}
	if pod.Status.Message != "" {
		return pod.Status.Message
	}
	return "pod " + strings.ToLower(string(pod.Status.Phase))
}

func (r *NotebookReconciler) setStatus(ctx context.Context, nb *notebooksv1alpha1.Notebook, phase, url, msg string) error {
	want := notebooksv1alpha1.NotebookStatus{Phase: phase, URL: url, Message: msg}
	if nb.Status == want {
		return nil
	}
	// Merge patch: no resourceVersion, so a stale cached copy cannot conflict.
	base := nb.DeepCopy()
	nb.Status = want
	return r.Status().Patch(ctx, nb, client.MergeFrom(base))
}

// component=singleuser-server is what the federation NetworkPolicies select.
var notebookLabels = map[string]string{
	"app.kubernetes.io/managed-by": "mip-notebook-operator",
	"component":                    "singleuser-server",
}

// pvcName is KubeSpawner's claim-{user_server} for a user's default server,
// so homes KubeSpawner created are found again. The slug functions below port
// its "safe" scheme (kubespawner/slugs.py, 7.1): a name kept as-is never
// contains "--", so it cannot equal an escaped one ending in "---<hash>".
func pvcName(user string) string {
	slug := safeSlug(user)
	if len(slug)+2 > slugMaxLength {
		slug = multiSlug(user, "")
	}
	return "claim-" + slug
}

const slugMaxLength = 48

var (
	nonAlnum   = regexp.MustCompile(`[^a-z0-9]+`)
	objectName = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)
)

func safeSlug(name string) string {
	if !strings.Contains(name, "--") && len(name) <= slugMaxLength && objectName.MatchString(name) {
		return name
	}
	return safeName(name, slugMaxLength-11) + "---" + shortHash(name)
}

// multiSlug hashes all names together; "\xff" never occurs in UTF-8, so
// different splits of the same characters hash differently.
func multiSlug(names ...string) string {
	limit := (slugMaxLength-9)/len(names) - 2
	slugs := make([]string, len(names))
	for i, n := range names {
		slugs[i] = safeName(n, limit)
	}
	return strings.Join(slugs, "--") + "---" + shortHash(strings.Join(names, "\xff"))
}

func safeName(name string, limit int) string {
	s := strings.TrimLeft(nonAlnum.ReplaceAllString(strings.ToLower(name), "-"), "-")
	s = strings.TrimRight(s[:min(len(s), limit)], "-")
	if s != "" && (s[0] < 'a' || s[0] > 'z') {
		s = "x-" + s[:min(len(s), limit-2)]
	}
	if s == "" {
		s = "x"
	}
	return s
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotebookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&notebooksv1alpha1.Notebook{}).
		Owns(&corev1.Pod{}).
		Named("notebook").
		Complete(r)
}
