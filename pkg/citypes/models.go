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
	GroupData = map[string]any
)

const NODE_GROUP_NAME string = "##groups"
