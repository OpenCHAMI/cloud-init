package cloud_config

import "testing"

func TestIsEmptyCloudConfig(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		content     string
		want        bool
	}{
		{
			name:        "empty part - header only",
			contentType: "text/cloud-config",
			content:     "#cloud-config\n",
			want:        true,
		},
		{
			name:        "empty part - header with blanks",
			contentType: "text/cloud-config",
			content:     "#cloud-config\n\n   \n",
			want:        true,
		},
		{
			name:        "non-empty part - has write_files",
			contentType: "text/cloud-config",
			content:     "#cloud-config\nwrite_files:\n  - path: /tmp/foo\n",
			want:        false,
		},
		{
			name:        "wrong content type",
			contentType: "text/x-shellscript",
			content:     "#cloud-config\n",
			want:        false,
		},
		{
			name:        "content type with charset param",
			contentType: "text/cloud-config; charset=utf-8",
			content:     "#cloud-config\n",
			want:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsEmptyCloudConfig(tc.contentType, tc.content)
			if got != tc.want {
				t.Errorf("IsEmptyCloudConfig(%q, ...) = %v, want %v", tc.contentType, got, tc.want)
			}
		})
	}
}
