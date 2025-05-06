package cmdline

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	cl := New()
	if cl == nil {
		t.Fatal("New() returned nil")
	}
	if len(cl.params) != 0 {
		t.Errorf("New() params map should be empty, got %d entries", len(cl.params))
	}
	if len(cl.flags) != 0 {
		t.Errorf("New() flags map should be empty, got %d entries", len(cl.flags))
	}
	if len(cl.order) != 0 {
		t.Errorf("New() order slice should be empty, got %d entries", len(cl.order))
	}
}

func TestAddParam(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "valid param",
			key:     "console",
			value:   "ttyS0",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			value:   "value",
			wantErr: true,
		},
		{
			name:    "empty value",
			key:     "key",
			value:   "",
			wantErr: true,
		},
		{
			name:    "key with spaces",
			key:     "key with spaces",
			value:   "value",
			wantErr: true,
		},
		{
			name:    "value with spaces",
			key:     "key",
			value:   "value with spaces",
			wantErr: true,
		},
		{
			name:    "key with quotes",
			key:     `key"with"quotes`,
			value:   "value",
			wantErr: true,
		},
		{
			name:    "value with quotes",
			key:     "key",
			value:   `value"with"quotes`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := New()
			err := cl.AddParam(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if val, ok := cl.params[tt.key]; !ok || val != tt.value {
					t.Errorf("AddParam() failed to add param, got %v, want %v", val, tt.value)
				}
			}
		})
	}
}

func TestAddParamDuplicate(t *testing.T) {
	cl := New()
	key := "console"
	value1 := "ttyS0"
	value2 := "ttyS1"

	if err := cl.AddParam(key, value1); err != nil {
		t.Fatalf("First AddParam() failed: %v", err)
	}

	err := cl.AddParam(key, value2)
	if err == nil {
		t.Error("AddParam() with duplicate key should return error")
	}
	if val := cl.params[key]; val != value1 {
		t.Errorf("AddParam() with duplicate key should not change value, got %v, want %v", val, value1)
	}
}

func TestAddFlag(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		wantErr bool
	}{
		{
			name:    "valid flag",
			flag:    "quiet",
			wantErr: false,
		},
		{
			name:    "valid flag with hyphen",
			flag:    "no-fb",
			wantErr: false,
		},
		{
			name:    "empty flag",
			flag:    "",
			wantErr: true,
		},
		{
			name:    "flag with spaces",
			flag:    "flag with spaces",
			wantErr: true,
		},
		{
			name:    "flag with quotes",
			flag:    `flag"with"quotes`,
			wantErr: true,
		},
		{
			name:    "flag as key=value",
			flag:    "console=ttyS0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := New()
			err := cl.AddFlag(tt.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if strings.Contains(tt.flag, "=") {
					parts := strings.SplitN(tt.flag, "=", 2)
					if val, ok := cl.params[parts[0]]; !ok || val != parts[1] {
						t.Errorf("AddFlag() failed to add param, got %v, want %v", val, parts[1])
					}
				} else {
					if _, ok := cl.flags[tt.flag]; !ok {
						t.Errorf("AddFlag() failed to add flag %v", tt.flag)
					}
				}
			}
		})
	}
}

func TestAddFlagDuplicate(t *testing.T) {
	cl := New()
	flag := "quiet"

	if err := cl.AddFlag(flag); err != nil {
		t.Fatalf("First AddFlag() failed: %v", err)
	}

	err := cl.AddFlag(flag)
	if err == nil {
		t.Error("AddFlag() with duplicate flag should return error")
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		flags    []string
		expected string
	}{
		{
			name: "only params",
			params: map[string]string{
				"console": "ttyS0",
				"root":    "/dev/sda1",
			},
			flags:    []string{},
			expected: "root=/dev/sda1 console=ttyS0",
		},
		{
			name:     "only flags",
			params:   map[string]string{},
			flags:    []string{"splash", "quiet"},
			expected: "quiet splash",
		},
		{
			name: "mixed params and flags",
			params: map[string]string{
				"console": "ttyS0",
				"root":    "/dev/sda1",
			},
			flags:    []string{"splash", "quiet"},
			expected: "root=/dev/sda1 quiet splash console=ttyS0",
		},
		{
			name: "initrd and root first",
			params: map[string]string{
				"console": "ttyS0",
				"root":    "/dev/sda1",
				"initrd":  "/boot/initrd.img",
			},
			flags:    []string{"quiet"},
			expected: "initrd=/boot/initrd.img root=/dev/sda1 quiet console=ttyS0",
		},
		{
			name: "multiple params in alphabetical order",
			params: map[string]string{
				"console": "ttyS0",
				"root":    "/dev/sda1",
				"initrd":  "/boot/initrd.img",
				"panic":   "30",
			},
			flags:    []string{"quiet", "splash"},
			expected: "initrd=/boot/initrd.img root=/dev/sda1 quiet splash console=ttyS0 panic=30",
		},
		{
			name:     "empty",
			params:   map[string]string{},
			flags:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := New()

			// Add params
			for k, v := range tt.params {
				if err := cl.AddParam(k, v); err != nil {
					t.Fatalf("AddParam() failed for %s=%s: %v", k, v, err)
				}
			}

			// Add flags
			for _, f := range tt.flags {
				if err := cl.AddFlag(f); err != nil {
					t.Fatalf("AddFlag() failed: %v", err)
				}
			}

			got := cl.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
