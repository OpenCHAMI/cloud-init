// Package cloud_config provides helpers for validating cloud-config content
// before it is served to nodes.
package cloud_config

import "strings"

// IsEmptyCloudConfig reports whether content is a text/cloud-config part
// that contains no module directives — i.e. it is only the '#cloud-config'
// header line plus optional blank lines/comments.
//
// Such empty parts cause cloud-init on the node to log warnings and can
// trigger TypeErrors in module handlers that expect a non-None config
// value (see issue #100).
func IsEmptyCloudConfig(contentType, content string) bool {
	if !strings.HasPrefix(strings.TrimSpace(contentType), "text/cloud-config") {
		return false
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip blank lines and the mandatory #cloud-config marker.
		if trimmed == "" || trimmed == "#cloud-config" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Found at least one non-comment, non-blank line — not empty.
		return false
	}
	return true
}
