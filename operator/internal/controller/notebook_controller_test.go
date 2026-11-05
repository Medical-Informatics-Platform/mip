package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	notebooksv1alpha1 "github.com/Medical-Informatics-Platform/mip/operator/api/v1alpha1"
)

var _ = Describe("Notebook controller", func() {
	var (
		ns         string
		reconciler *NotebookReconciler
		counter    int
	)

	newNotebook := func(name string, env map[string]string) *notebooksv1alpha1.Notebook {
		return &notebooksv1alpha1.Notebook{
			ObjectMeta: metav1.ObjectMeta{Name: "jupyter-" + name, Namespace: ns},
			Spec: notebooksv1alpha1.NotebookSpec{
				Profile: "default",
				User:    "Alice@Example.org",
				Env:     env,
			},
		}
	}
	newProfile := func() *notebooksv1alpha1.NotebookProfile {
		return &notebooksv1alpha1.NotebookProfile{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ns},
			Spec: notebooksv1alpha1.NotebookProfileSpec{
				Image:   "hbpmip/mip-jupyter:test",
				Storage: notebooksv1alpha1.NotebookStorage{Size: resource.MustParse("2Gi"), StorageClassName: "fast"},
				Env:     []corev1.EnvVar{{Name: "PLATFORM_BACKEND_URL", Value: "http://backend"}},
			},
		}
	}
	reconcileNB := func(nb *notebooksv1alpha1.Notebook) reconcile.Result {
		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(nb)})
		Expect(err).NotTo(HaveOccurred())
		return res
	}
	fetch := func(nb *notebooksv1alpha1.Notebook) *notebooksv1alpha1.Notebook {
		out := &notebooksv1alpha1.Notebook{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(nb), out)).To(Succeed())
		return out
	}

	BeforeEach(func() {
		counter++
		ns = fmt.Sprintf("federation-%d", counter)
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})).To(Succeed())
		reconciler = &NotebookReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
	})

	Context("CRD validation", func() {
		It("rejects env names outside JUPYTERHUB_* / JPY_*", func() {
			err := k8sClient.Create(ctx, newNotebook("bad-env", map[string]string{"LD_PRELOAD": "/x.so"}))
			Expect(apierrors.IsInvalid(err)).To(BeTrue(), "got %v", err)
		})
		It("rejects the API token in env", func() {
			err := k8sClient.Create(ctx, newNotebook("token-env", map[string]string{"JUPYTERHUB_API_TOKEN": "x"}))
			Expect(apierrors.IsInvalid(err)).To(BeTrue(), "got %v", err)
		})
		It("accepts only jupyter-* names of at most 57 characters", func() {
			nb := newNotebook("x", nil)
			nb.Name = "keycloak-credentials"
			Expect(apierrors.IsInvalid(k8sClient.Create(ctx, nb))).To(BeTrue())
			Expect(apierrors.IsInvalid(k8sClient.Create(ctx, newNotebook(strings.Repeat("a", 50), nil)))).To(BeTrue())
			Expect(k8sClient.Create(ctx, newNotebook(strings.Repeat("a", 49), nil))).To(Succeed())
		})
		It("rejects spec changes", func() {
			nb := newNotebook("immutable", nil)
			Expect(k8sClient.Create(ctx, nb)).To(Succeed())
			nb.Spec.Profile = "other"
			Expect(apierrors.IsInvalid(k8sClient.Update(ctx, nb))).To(BeTrue())
		})
	})

	It("fails when the profile is missing", func() {
		nb := newNotebook("no-profile", nil)
		Expect(k8sClient.Create(ctx, nb)).To(Succeed())
		Expect(reconcileNB(nb).RequeueAfter).To(Equal(profileRetryWait))
		got := fetch(nb)
		Expect(got.Status.Phase).To(Equal(notebooksv1alpha1.PhaseFailed))
		Expect(got.Status.Message).To(ContainSubstring(`"default" not found`))
	})

	It("creates the PVC and a restricted pod, then reports Running", func() {
		Expect(k8sClient.Create(ctx, newProfile())).To(Succeed())
		nb := newNotebook("happy", map[string]string{"JUPYTERHUB_SERVICE_PREFIX": "/notebook/user/alice/"})
		Expect(k8sClient.Create(ctx, nb)).To(Succeed())
		reconcileNB(nb)

		pvc := &corev1.PersistentVolumeClaim{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: pvcName(nb.Spec.User)}, pvc)).To(Succeed())
		Expect(pvc.OwnerReferences).To(BeEmpty())
		Expect(*pvc.Spec.StorageClassName).To(Equal("fast"))

		pod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(nb), pod)).To(Succeed())
		Expect(pod.OwnerReferences).To(HaveLen(1))
		Expect(pod.Labels).To(HaveKeyWithValue("component", "singleuser-server"))
		Expect(*pod.Spec.AutomountServiceAccountToken).To(BeFalse())
		Expect(*pod.Spec.SecurityContext.RunAsNonRoot).To(BeTrue())
		Expect(pod.Spec.InitContainers).To(BeEmpty())
		c := pod.Spec.Containers[0]
		Expect(c.Image).To(Equal("hbpmip/mip-jupyter:test"))
		// Defaulted by the CRD; buildPod has no fallback of its own.
		Expect(c.VolumeMounts[0].MountPath).To(Equal("/home/jovyan"))
		Expect(*c.SecurityContext.AllowPrivilegeEscalation).To(BeFalse())
		env := map[string]corev1.EnvVar{}
		for _, e := range c.Env {
			env[e.Name] = e
		}
		Expect(env["PLATFORM_BACKEND_URL"].Value).To(Equal("http://backend"))
		Expect(env["MIP_WORK_DIR"].Value).To(Equal("/home/jovyan/work"))
		Expect(env["JUPYTERHUB_SERVICE_PREFIX"].Value).To(Equal("/notebook/user/alice/"))
		Expect(env["JUPYTERHUB_API_TOKEN"].ValueFrom.SecretKeyRef.Name).To(Equal(nb.Name + "-" + string(nb.UID) + "-token"))
		Expect(env["JUPYTERHUB_API_TOKEN"].Value).To(BeEmpty())
		Expect(fetch(nb).Status.Phase).To(Equal(notebooksv1alpha1.PhasePending))

		pod.Status.PodIP = "10.0.0.7"
		pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
		Expect(k8sClient.Status().Update(ctx, pod)).To(Succeed())
		reconcileNB(nb)
		got := fetch(nb)
		Expect(got.Status.Phase).To(Equal(notebooksv1alpha1.PhaseRunning))
		Expect(got.Status.URL).To(Equal("http://10.0.0.7:8888"))
	})

	It("reports Failed when the notebook exits", func() {
		pod := &corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{ExitCode: 137, Reason: "OOMKilled"},
			}}},
		}}
		phase, url, msg := podStatus(pod)
		Expect(phase).To(Equal(notebooksv1alpha1.PhaseFailed))
		Expect(url).To(BeEmpty())
		Expect(msg).To(ContainSubstring("code 137: OOMKilled"))
	})

	It("adds the root init container only when asked", func() {
		profile := newProfile()
		profile.Spec.Storage.FixOwnership = true
		pod := buildPod(newNotebook("x", nil), profile)
		Expect(pod.Spec.InitContainers).To(HaveLen(1))
		Expect(pod.Spec.InitContainers[0].SecurityContext.RunAsUser).To(Equal(ptr.To[int64](0)))
	})

	Context("shared examples", func() {
		BeforeEach(func() {
			reconciler.ExamplesClaim = "mip-examples"
		})

		// No PV controller runs under envtest, so the test binds the claim itself.
		examplesClaim := func(bound bool) {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "mip-examples", Namespace: ns},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
			if bound {
				pvc.Status.Phase = corev1.ClaimBound
				Expect(k8sClient.Status().Update(ctx, pvc)).To(Succeed())
			}
		}
		// home "" keeps the CRD default, /home/jovyan.
		spawn := func(fixOwnership bool, home string) (*notebooksv1alpha1.Notebook, *corev1.Pod) {
			profile := newProfile()
			profile.Spec.Storage.FixOwnership = fixOwnership
			profile.Spec.Storage.MountPath = home
			Expect(k8sClient.Create(ctx, profile)).To(Succeed())
			nb := newNotebook("examples", nil)
			Expect(k8sClient.Create(ctx, nb)).To(Succeed())
			reconcileNB(nb)
			pod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(nb), pod)).To(Succeed())
			return nb, pod
		}

		It("mounts a bound claim read-only", func() {
			examplesClaim(true)
			_, pod := spawn(true, "")
			Expect(pod.Spec.Volumes).To(ContainElement(corev1.Volume{
				Name: "examples",
				VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "mip-examples",
					ReadOnly:  true,
				}},
			}))
			Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(
				corev1.VolumeMount{Name: "examples", MountPath: "/home/jovyan/work/.mip-examples", ReadOnly: true}))
		})
		It("mounts inside the profile's home", func() {
			examplesClaim(true)
			_, pod := spawn(true, "/home/user")
			Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(
				corev1.VolumeMount{Name: "examples", MountPath: "/home/user/work/.mip-examples", ReadOnly: true}))
			// The entrypoint looks for the mount under MIP_WORK_DIR.
			Expect(pod.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "MIP_WORK_DIR", Value: "/home/user/work"}))
		})
		It("starts without the mount when there is no claim", func() {
			nb, pod := spawn(true, "")
			Expect(pod.Spec.Volumes).To(HaveLen(1))
			Expect(pod.Spec.Volumes[0].Name).To(Equal(homeVolume))
			Expect(fetch(nb).Status.Phase).To(Equal(notebooksv1alpha1.PhasePending))
		})
		It("ignores a claim that is not bound", func() {
			examplesClaim(false)
			_, pod := spawn(true, "")
			Expect(pod.Spec.Volumes).To(HaveLen(1))
		})
		It("does nothing without EXAMPLES_CLAIM", func() {
			reconciler.ExamplesClaim = ""
			examplesClaim(true)
			_, pod := spawn(true, "")
			Expect(pod.Spec.Volumes).To(HaveLen(1))
		})
		It("needs fixOwnership, so work/ exists before the mount", func() {
			examplesClaim(true)
			_, pod := spawn(false, "")
			Expect(pod.Spec.Volumes).To(HaveLen(1))
		})
	})

	It("lets the hub create only Opaque <name>-<uid>-token Secrets owned by the Notebook", func() {
		policy, err := os.ReadFile(filepath.Join("..", "..", "testdata", "token-secrets-policy.yaml"))
		Expect(err).NotTo(HaveOccurred())
		for _, doc := range strings.Split(string(policy), "\n---\n") {
			obj := &unstructured.Unstructured{}
			Expect(utilyaml.Unmarshal([]byte(doc), &obj.Object)).To(Succeed())
			Expect(k8sClient.Create(ctx, obj)).To(Succeed())
		}
		// The hub Role from the madgik/mip chart, bound to the hub ServiceAccount.
		Expect(k8sClient.Create(ctx, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "jupyterhub", Namespace: ns},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{notebooksv1alpha1.GroupVersion.Group}, Resources: []string{"notebooks"}, Verbs: []string{"create"}},
				{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"create"}},
			},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "jupyterhub", Namespace: ns},
			Subjects:   []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Name: "jupyterhub", Namespace: ns}},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "jupyterhub"},
		})).To(Succeed())
		hubCfg := rest.CopyConfig(cfg)
		hubCfg.Impersonate.UserName = "system:serviceaccount:" + ns + ":jupyterhub"
		hub, err := client.New(hubCfg, client.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())

		nb := newNotebook("policy", nil)
		Expect(hub.Create(ctx, nb)).To(Succeed())
		secret := func(name string, typ corev1.SecretType) *corev1.Secret {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, OwnerReferences: []metav1.OwnerReference{{
					APIVersion: notebooksv1alpha1.GroupVersion.String(), Kind: "Notebook", Name: nb.Name, UID: nb.UID,
				}}},
				Type: typ,
			}
		}
		denied := MatchError(ContainSubstring("mip-jupyterhub-secrets"))
		// The policy takes effect a moment after it is created.
		Eventually(func() error {
			return hub.Create(ctx, secret("keycloak-credentials", corev1.SecretTypeOpaque))
		}).WithTimeout(10 * time.Second).Should(denied)
		own := nb.Name + "-" + string(nb.UID) + "-token"
		// The API server would fill this with the named ServiceAccount's token.
		saToken := secret(own, corev1.SecretTypeServiceAccountToken)
		saToken.Annotations = map[string]string{corev1.ServiceAccountNameKey: "default"}
		Expect(hub.Create(ctx, saToken)).To(denied)
		Expect(hub.Create(ctx, secret(own, corev1.SecretTypeOpaque))).To(Succeed())
	})
})

var _ = Describe("names", func() {
	It("matches KubeSpawner's claim-{user_server}", func() {
		// From KubeSpawner 7.1 (slug_scheme "safe") in the hub image.
		for user, want := range map[string]string{
			"alice":                                "claim-alice",
			"john.doe":                             "claim-john-doe---30f69670",
			"john-doe-30f69670":                    "claim-john-doe-30f69670", // must not equal john.doe's
			"Alice":                                "claim-alice---3bc51062",
			"alice--x":                             "claim-alice-x---2bbc534b",
			"Alice@Example.org":                    "claim-alice-example-org---cfd44c4f",
			"3b64fbf7-4e7b-4455-a61e-c19e64905f06": "claim-x-3b64fbf7-4e7b-4--x---07e56cda",
			strings.Repeat("a", 46):                "claim-" + strings.Repeat("a", 46),
			strings.Repeat("a", 47):                "claim-aaaaaaaaaaaaaaaaa--x---b191b19d",
			strings.Repeat("x.y", 20):              "claim-x-yx-yx-yx-yx-yx--x---9a32d384",
			"user-":                                "claim-user---b5d4a457",
			"_":                                    "claim-x---d2e2adf7",
			"1":                                    "claim-x-1---6b86b273",
		} {
			Expect(pvcName(user)).To(Equal(want), user)
		}
	})
})
