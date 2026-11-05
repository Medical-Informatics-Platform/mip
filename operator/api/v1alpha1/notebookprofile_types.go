package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NotebookProfileSpec is the pod template for notebooks. It is shipped with the
// chart; the hub cannot read or change it. The security context is not part of
// it on purpose: the operator always applies the restricted one.
type NotebookProfileSpec struct {
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// +optional
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Storage NotebookStorage `json:"storage"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Env is static, non-secret configuration for every notebook.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// NotebookStorage describes the per-user home volume.
type NotebookStorage struct {
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	Size resource.Quantity `json:"size"`

	// +optional
	// +kubebuilder:default=/home/jovyan
	MountPath string `json:"mountPath,omitempty"`

	// FixOwnership adds a root init container that chowns the volume to
	// 1000:100, for provisioners that ignore fsGroup (local-path, hostPath).
	// +optional
	FixOwnership bool `json:"fixOwnership,omitempty"`
}

// +kubebuilder:object:root=true

// NotebookProfile is a pod template that Notebooks reference by name.
type NotebookProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NotebookProfileSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// NotebookProfileList contains a list of NotebookProfile.
type NotebookProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NotebookProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NotebookProfile{}, &NotebookProfileList{})
}
