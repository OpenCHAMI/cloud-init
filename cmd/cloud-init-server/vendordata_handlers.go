package main

import "net/http"

// VendorDataHandler godoc
// @Summary Get vendor data
// @Description For OpenCHAMI, the vendor-data will always be a list of other #cloud-config URLs to download and merge.
// @Produce plain
// @Success 200 {string} string
// @Router /vendor-data [get]
func VendorDataHandler(w http.ResponseWriter, r *http.Request) {
	payload := `#template: jinja
#include
{% for group_name in vendor_data.groups.keys() %}
https://{{ vendor_data.cloud_init_base_url }}/{{ group_name }}.yaml
{% endfor %}
`
	w.Write([]byte(payload))
}
