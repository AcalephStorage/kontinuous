package kube

type Secret struct {
	Kind       string                 `json:"apiVersion",omitempty"`
	ApiVersion string                 `json:"apiVersion,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Data       map[string]string      `json:"data,omitempty"`
	Type       string                 `json:"type, omitempty"`
}
