package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// QuickstartFileName default name of the source repository configuration
	QuickstartFileName = "source-config.yaml"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Quickstart represents a collection quickstart project
//
// +k8s:openapi-gen=true
type Quickstart struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the Quickstart from the client
	// +optional
	Spec QuickstartSpec `json:"spec"`
}

// QuickstartList contains a list of Quickstarts
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type QuickstartList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Quickstart `json:"items"`
}

// QuickstartSpec defines the desired state of Quickstart.
type QuickstartSpec struct {
	// Quickstarts the quickstart sources
	Quickstarts []QuickstartSource `json:"quickstarts,omitempty"`
}

// QuickstartSource the source of a quickstart
type QuickstartSource struct {
	ID             string
	Owner          string
	Name           string
	Version        string
	Language       string
	Framework      string
	Tags           []string
	DownloadZipURL string
	GitServer      string
	GitKind        string
}
