package bss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootParams_ValidateBootParams(t *testing.T) {
	tests := []struct {
		name    string
		params  *BootParams
		wantErr bool
	}{
		{
			name: "valid boot params",
			params: &BootParams{
				Kernel: "http://example.com/kernel",
				Initrd: "http://example.com/initrd",
			},
			wantErr: false,
		},
		{
			name: "missing kernel",
			params: &BootParams{
				Initrd: "http://example.com/initrd",
			},
			wantErr: true,
		},
		{
			name: "missing initrd",
			params: &BootParams{
				Kernel: "http://example.com/kernel",
			},
			wantErr: true,
		},
		{
			name:    "empty boot params",
			params:  &BootParams{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.ValidateBootParams()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeBootParams(t *testing.T) {
	tests := []struct {
		name     string
		params   []*BootParams
		expected *BootParams
		wantErr  bool
	}{
		{
			name: "merge basic params",
			params: []*BootParams{
				{
					Kernel: "http://example.com/kernel1",
					Initrd: "http://example.com/initrd1",
					Params: "console=ttyS0",
				},
				{
					Kernel: "http://example.com/kernel2",
					Initrd: "http://example.com/initrd2",
					Params: "root=/dev/sda1",
				},
			},
			expected: &BootParams{
				Kernel: "http://example.com/kernel1",
				Initrd: "http://example.com/initrd1",
				Params: "console=ttyS0 root=/dev/sda1",
			},
			wantErr: false,
		},
		{
			name: "merge with rootfs",
			params: []*BootParams{
				{
					Kernel: "http://example.com/kernel",
					Initrd: "http://example.com/initrd",
					RootFS: &RootFS{
						Type:    "nfs",
						Server:  "10.0.0.1",
						Path:    "/nfsroot",
						Options: "vers=4,ro",
					},
				},
				{
					Kernel: "http://example.com/kernel2",
					Initrd: "http://example.com/initrd2",
					Params: "console=ttyS0",
				},
			},
			expected: &BootParams{
				Kernel: "http://example.com/kernel",
				Initrd: "http://example.com/initrd",
				Params: "console=ttyS0",
				RootFS: &RootFS{
					Type:    "nfs",
					Server:  "10.0.0.1",
					Path:    "/nfsroot",
					Options: "vers=4,ro",
				},
			},
			wantErr: false,
		},
		{
			name: "merge with cloud-init",
			params: []*BootParams{
				{
					Kernel: "http://example.com/kernel",
					Initrd: "http://example.com/initrd",
					CloudInit: &CloudInitServer{
						URL:     "http://cloud-init.example.com",
						Version: "v1",
					},
				},
				{
					Kernel: "http://example.com/kernel2",
					Initrd: "http://example.com/initrd2",
					Params: "console=ttyS0",
				},
			},
			expected: &BootParams{
				Kernel: "http://example.com/kernel",
				Initrd: "http://example.com/initrd",
				Params: "console=ttyS0",
				CloudInit: &CloudInitServer{
					URL:     "http://cloud-init.example.com",
					Version: "v1",
				},
			},
			wantErr: false,
		},
		{
			name: "merge with all components",
			params: []*BootParams{
				{
					Kernel: "http://example.com/kernel",
					Initrd: "http://example.com/initrd",
					Params: "console=ttyS0",
					RootFS: &RootFS{
						Type:   "nfs",
						Server: "10.0.0.1",
						Path:   "/nfsroot",
					},
					CloudInit: &CloudInitServer{
						URL:     "http://cloud-init.example.com",
						Version: "v1",
					},
				},
				{
					Kernel: "http://example.com/kernel2",
					Initrd: "http://example.com/initrd2",
					Params: "root=/dev/sda1",
				},
			},
			expected: &BootParams{
				Kernel: "http://example.com/kernel",
				Initrd: "http://example.com/initrd",
				Params: "console=ttyS0 root=/dev/sda1",
				RootFS: &RootFS{
					Type:   "nfs",
					Server: "10.0.0.1",
					Path:   "/nfsroot",
				},
				CloudInit: &CloudInitServer{
					URL:     "http://cloud-init.example.com",
					Version: "v1",
				},
			},
			wantErr: false,
		},
		{
			name: "merge empty params",
			params: []*BootParams{
				{},
				{},
			},
			expected: &BootParams{},
			wantErr:  false,
		},
		{
			name:     "merge nil params",
			params:   nil,
			expected: &BootParams{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergeBootParams(tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Kernel, merged.Kernel)
			assert.Equal(t, tt.expected.Initrd, merged.Initrd)
			assert.Equal(t, tt.expected.Params, merged.Params)

			if tt.expected.RootFS != nil {
				assert.NotNil(t, merged.RootFS)
				assert.Equal(t, tt.expected.RootFS.Type, merged.RootFS.Type)
				assert.Equal(t, tt.expected.RootFS.Server, merged.RootFS.Server)
				assert.Equal(t, tt.expected.RootFS.Path, merged.RootFS.Path)
				assert.Equal(t, tt.expected.RootFS.Options, merged.RootFS.Options)
			} else {
				assert.Nil(t, merged.RootFS)
			}

			if tt.expected.CloudInit != nil {
				assert.NotNil(t, merged.CloudInit)
				assert.Equal(t, tt.expected.CloudInit.URL, merged.CloudInit.URL)
				assert.Equal(t, tt.expected.CloudInit.Version, merged.CloudInit.Version)
			} else {
				assert.Nil(t, merged.CloudInit)
			}
		})
	}
}

func TestBootParams_ParseFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid json",
			json: `{
				"kernel": "http://example.com/kernel",
				"initrd": "http://example.com/initrd",
				"params": "console=ttyS0"
			}`,
			wantErr: false,
		},
		{
			name: "missing kernel",
			json: `{
				"initrd": "http://example.com/initrd"
			}`,
			wantErr: true,
		},
		{
			name: "missing initrd",
			json: `{
				"kernel": "http://example.com/kernel"
			}`,
			wantErr: true,
		},
		{
			name:    "empty json",
			json:    `{}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &BootParams{}
			err := params.ParseFromJSON([]byte(tt.json))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
