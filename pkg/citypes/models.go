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
	GroupData = map[string]any
)

// NOTE: This is just a unique constant value for access group data being stored
// as a citypes.CI since the API's require an IDENTIFIER to access.
//
// NOTE: This may be removed later after creating a separate group structure.
const GROUP_IDENTIFIER string = "%%groups%%"
