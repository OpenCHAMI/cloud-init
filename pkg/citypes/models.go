package citypes

type CI struct {
	Name   string `json:"name"`
	CIData CIData `json:"cloud-init"`
}

type CIData struct {
	UserData   map[string]any `json:"user-data"`
	MetaData   map[string]any `json:"meta-data"`
	VendorData map[string]any `json:"vendor-data"`
}

type (
	// only defined for readibility
	UserData = map[string]any
)

type WriteFiles struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Group   string `json:"group,omitempty"`
}

type GroupData struct {
	Name    string         `json:"name,omitempty"`
	Data    MetaDataKV     `json:"meta-data,omitempty"`
	Actions map[string]any `json:"user-data,omitempty"`
}

type MetaDataKV map[string]string // Metadata for the group may only contain key value pairs
