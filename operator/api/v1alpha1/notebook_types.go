package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Notebook phases reported in status.phase.
const (
	PhasePending = "Pending"
	PhaseRunning = "Running"
	PhaseFailed  = "Failed"
)

// NotebookSpec is what JupyterHub asks for. Everything that decides what runs
// (image, resources, volumes, security) comes from the referenced profile.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable; delete and recreate the Notebook"
type NotebookSpec struct {
	// Profile is the name of a NotebookProfile in the same namespace.
	// +kubebuilder:validation:MinLength=1
	Profile string `json:"profile"`

	// User is the JupyterHub user name. It names the home PVC.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	User string `json:"user"`

	// ServerName is the JupyterHub named server, empty for the default server.
	// +optional
	ServerName string `json:"serverName,omitempty"`

	// Env holds the per-spawn JupyterHub variables. Only JUPYTERHUB_* and JPY_*
	// names are accepted; the API token comes from the Notebook's token Secret.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.all(k, k.matches('^(JUPYTERHUB|JPY)_[A-Z0-9_]+$'))",message="env keys must match ^(JUPYTERHUB|JPY)_[A-Z0-9_]+$"
	// +kubebuilder:validation:XValidation:rule="!('JUPYTERHUB_API_TOKEN' in self) && !('JPY_API_TOKEN' in self)",message="the API token comes from the Notebook's token Secret"
	Env map[string]string `json:"env,omitempty"`
}

// NotebookStatus is the observed state of the notebook pod.
type NotebookStatus struct {
	// +optional
	// +kubebuilder:validation:Enum=Pending;Running;Failed
	Phase string `json:"phase,omitempty"`

	// URL is where the hub proxy reaches the notebook server, set when Running.
	// +optional
	URL string `json:"url,omitempty"`

	// Message is the latest human readable reason, shown on the spawn page.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="User",type=string,JSONPath=`.spec.user`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Notebook is one JupyterHub single-user server. The hub creates Secret
// <name>-<uid>-token (key "token", owned by the Notebook) with its API token.
// The API server assigns the UID, so no Secret that existed before the
// Notebook, such as one left by an earlier Notebook of the same name, is ever
// mounted. Names are jupyter-* and at most 57 characters, as the hub makes them.
// +kubebuilder:validation:XValidation:rule="self.metadata.name.startsWith('jupyter-') && size(self.metadata.name) <= 57",message="name must start with jupyter- and be at most 57 characters"
type Notebook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotebookSpec   `json:"spec"`
	Status NotebookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NotebookList contains a list of Notebook.
type NotebookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notebook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notebook{}, &NotebookList{})
}
