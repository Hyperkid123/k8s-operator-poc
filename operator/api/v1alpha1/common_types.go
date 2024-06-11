package v1alpha1

type RouteDefinition struct {
	Pathname   string `json:"pathname"`
	Exact      bool   `json:"exact,omitempty"`
	Module     string `json:"module"`
	ImportName string `json:"importName,omitempty"`
	// +kubebuilder:validation:XPreserveUnknownFields
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	Props map[string]string `json:"props,omitempty"`
}

type UiModule struct {
	ManifestLocation string            `json:"manifestLocation"`
	Routes           []RouteDefinition `json:"routes"`
}
