package bss

import (
	"fmt"
	"strings"
)

// GenerateBootScript creates an iPXE boot script from the given boot parameters
func GenerateBootScript(params *BootParams, retry int, arch string) (string, error) {
	if params.Kernel == "" {
		return "", fmt.Errorf("kernel must be specified to generate boot script")
	}

	var script strings.Builder
	script.WriteString("#!ipxe\n")

	// Build kernel command line
	kernelCmd := fmt.Sprintf("kernel --name kernel %s", params.Kernel)

	// Add root filesystem parameters if specified
	if params.RootFS != nil {
		switch params.RootFS.Type {
		case "nfs":
			// NFS root requires network configuration
			kernelCmd += " rd.neednet=1"
			if params.RootFS.Server != "" && params.RootFS.Path != "" {
				kernelCmd += fmt.Sprintf(" root=nfs://%s:%s", params.RootFS.Server, params.RootFS.Path)
				if params.RootFS.Options != "" {
					kernelCmd += fmt.Sprintf(" rootflags=%s", params.RootFS.Options)
				}
			}
		case "local":
			// Local root filesystem
			if params.RootFS.Path != "" {
				kernelCmd += fmt.Sprintf(" root=%s", params.RootFS.Path)
				if params.RootFS.Options != "" {
					kernelCmd += fmt.Sprintf(" rootflags=%s", params.RootFS.Options)
				}
			}
		}
	}

	// Add cloud-init server configuration if specified
	if params.CloudInit != nil && params.CloudInit.URL != "" {
		kernelCmd += fmt.Sprintf(" ds=nocloud-net;s=%s", params.CloudInit.URL)
		if params.CloudInit.Version != "" {
			kernelCmd += fmt.Sprintf(";v=%s", params.CloudInit.Version)
		}
	}

	// Add any additional parameters
	if params.Params != "" {
		kernelCmd += " " + params.Params
	}

	script.WriteString(kernelCmd + " || goto boot_retry\n")

	// Add initrd if specified
	if params.Initrd != "" {
		script.WriteString(fmt.Sprintf("initrd --name initrd %s || goto boot_retry\n", params.Initrd))
	}

	// Add boot command
	script.WriteString("boot || goto boot_retry\n")

	// Add retry section
	script.WriteString(":boot_retry\n")
	script.WriteString("sleep 30\n")

	// Add retry chain command
	retryParams := make([]string, 0)
	if retry > 0 {
		retryParams = append(retryParams, fmt.Sprintf("retry=%d", retry))
	}
	if arch != "" {
		retryParams = append(retryParams, fmt.Sprintf("arch=%s", arch))
	}

	script.WriteString(fmt.Sprintf("chain https://api-gw-service-nmn.local/apis/bss/boot/v1/bootscript?%s\n",
		strings.Join(retryParams, "&")))

	return script.String(), nil
}
