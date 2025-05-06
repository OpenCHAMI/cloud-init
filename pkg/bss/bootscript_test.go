package bss

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateBootScript_Basic(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		Params: "console=ttyS0,115200 root=/dev/sda1",
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)
	assert.Contains(t, script, "#!ipxe")
	assert.Contains(t, script, "kernel --name kernel http://example.com/kernel root=/dev/sda1 console=ttyS0,115200 ")
	assert.Contains(t, script, "initrd --name initrd http://example.com/initrd")
	assert.Contains(t, script, "boot || goto boot_retry")
	assert.Contains(t, script, ":boot_retry")
	assert.Contains(t, script, "sleep 30")
	assert.Contains(t, script, "chain https://api-gw-service-nmn.local/apis/bss/boot/v1/bootscript")
}

func TestGenerateBootScript_WithRetry(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
	}

	script, err := GenerateIPXEScript(params, 3, "")
	assert.NoError(t, err)
	assert.Contains(t, script, "retry=3")
}

func TestGenerateBootScript_WithArch(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
	}

	script, err := GenerateIPXEScript(params, 0, "x86_64")
	assert.NoError(t, err)
	assert.Contains(t, script, "arch=x86_64")
}

func TestGenerateBootScript_WithOptions(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		Params: "console=ttyS0,115200 root=/dev/sda1",
	}

	script, err := GenerateIPXEScript(params, 2, "x86_64")
	assert.NoError(t, err)

	// Check all components are present
	assert.Contains(t, script, "kernel --name kernel http://example.com/kernel root=/dev/sda1 console=ttyS0,115200")
	assert.Contains(t, script, "initrd --name initrd http://example.com/initrd")
	assert.Contains(t, script, "boot || goto boot_retry")
	assert.Contains(t, script, "retry=2")
	assert.Contains(t, script, "arch=x86_64")
}

func TestGenerateBootScript_ErrorCases(t *testing.T) {
	// Test missing kernel
	params := &BootParams{
		Params: "console=ttyS0,115200",
	}

	_, err := GenerateIPXEScript(params, 0, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kernel must be specified")

	// Test empty params
	params = &BootParams{}
	_, err = GenerateIPXEScript(params, 0, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kernel must be specified")

}

func TestGenerateBootScript_RetryChainURL(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
	}

	script, err := GenerateIPXEScript(params, 1, "x86_64")
	assert.NoError(t, err)

	// Verify the retry chain URL is properly formatted
	expectedURL := "chain https://api-gw-service-nmn.local/apis/bss/boot/v1/bootscript?retry=1&arch=x86_64"
	assert.Contains(t, script, expectedURL)
}

func TestGenerateBootScript_Ordering(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		Params: "console=ttyS0,115200",
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)

	// Split the script into lines to properly check command ordering
	lines := strings.Split(script, "\n")

	// Find the line numbers for each command
	var kernelLine, initrdLine, bootLine, retryLine int
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "kernel"):
			kernelLine = i
		case strings.HasPrefix(line, "initrd"):
			initrdLine = i
		case strings.HasPrefix(line, "boot"):
			bootLine = i
		case strings.HasPrefix(line, ":boot_retry"):
			retryLine = i
		}
	}

	// Verify the order of commands
	assert.True(t, kernelLine < initrdLine, "kernel command should come before initrd")
	assert.True(t, initrdLine < bootLine, "initrd command should come before boot")
	assert.True(t, bootLine < retryLine, "boot command should come before retry section")
}

func TestGenerateBootScript_WithNFSRoot(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		RootFS: &RootFS{
			Type:    "nfs",
			Server:  "10.0.0.1",
			Path:    "/nfsroot",
			Options: "vers=4,ro",
		},
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)

	// Verify NFS root parameters
	assert.Contains(t, script, "rd.neednet=1")
	assert.Contains(t, script, "root=nfs://10.0.0.1:/nfsroot")
	assert.Contains(t, script, "rootflags=vers=4,ro")

	// Verify remote initrd loading
	assert.Contains(t, script, "initrd --name initrd http://example.com/initrd")
}

func TestGenerateBootScript_WithLocalRoot(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		RootFS: &RootFS{
			Type:    "local",
			Path:    "/dev/sda1",
			Options: "ro,noatime",
		},
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)

	// Verify local root parameters
	assert.Contains(t, script, "root=/dev/sda1")
	assert.Contains(t, script, "rootflags=ro,noatime")

	// Verify remote initrd loading
	assert.Contains(t, script, "initrd --name initrd http://example.com/initrd")
}

func TestGenerateBootScript_WithRootFSAndParams(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		RootFS: &RootFS{
			Type:   "nfs",
			Server: "10.0.0.1",
			Path:   "/nfsroot",
		},
		Params: "console=ttyS0,115200",
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)

	// Verify both rootfs and additional parameters are included
	assert.Contains(t, script, "root=nfs://10.0.0.1:/nfsroot")
	assert.Contains(t, script, "console=ttyS0,115200")
}

func TestGenerateBootScript_WithCloudInit(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		CloudInit: &CloudInitServer{
			URL:     "http://cloud-init.example.com",
			Version: "v1",
		},
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)

	// Verify cloud-init parameters
	assert.Contains(t, script, "ds=nocloud-net;s=http://cloud-init.example.com;v=v1")
}

func TestGenerateBootScript_WithCloudInitAndRootFS(t *testing.T) {
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		RootFS: &RootFS{
			Type:   "nfs",
			Server: "10.0.0.1",
			Path:   "/nfsroot",
		},
		CloudInit: &CloudInitServer{
			URL: "http://cloud-init.example.com",
		},
	}

	script, err := GenerateIPXEScript(params, 0, "")
	assert.NoError(t, err)

	// Verify both rootfs and cloud-init parameters
	assert.Contains(t, script, "root=nfs://10.0.0.1:/nfsroot")
	assert.Contains(t, script, "ds=nocloud-net;s=http://cloud-init.example.com")
}
