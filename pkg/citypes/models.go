package citypes

type CI struct {
	Name   string `json:"name"`
	CIData CIData `json:"cloud-init"`
}

type CIData struct {
	UserData   map[string]interface{} `json:"userdata"`
	MetaData   map[string]interface{} `json:"metadata"`
	VendorData map[string]interface{} `json:"vendordata"`
}

type (
	// only defined for readibility
	UserData  = map[string]any
	Group     map[string]GroupData
	GroupData map[string]any
)

type WriteFiles struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Group   string `json:"group,omitempty"`
}
