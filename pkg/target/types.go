package target

type Target struct {
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
	Region   string `json:"region,omitempty"`
}
