package bss

import (
	"fmt"
	"strings"

	"github.com/OpenCHAMI/cloud-init/pkg/cmdline"
)

func GenerateIPXEScript(params *BootParams, retry int, arch string) (string, error) {
	if params.Kernel == "" {
		return "", fmt.Errorf("kernel must be specified")
	}
	if params.Initrd == "" {
		return "", fmt.Errorf("initrd must be specified")
	}

	var script strings.Builder
	script.WriteString("#!ipxe\n")

	// Build kernel command line
	kernelCmd := fmt.Sprintf("kernel --name kernel %s", params.Kernel)

	cmd := cmdline.New()
	for _, param := range strings.Split(params.Params, " ") {
		cmd.AddFlag(param)
	}

	// Add root filesystem parameters if specified
	if params.RootFS != nil {
		switch params.RootFS.Type {
		case "nfs":
			// NFS root requires network configuration
			kernelCmd += " rd.neednet=1"
			if params.RootFS.Server != "" && params.RootFS.Path != "" {
				cmd.AddParam("root", fmt.Sprintf("nfs://%s:%s", params.RootFS.Server, params.RootFS.Path))
				if params.RootFS.Options != "" {
					cmd.AddParam("rootflags", params.RootFS.Options)
				}
			}
		case "local":
			// Local root filesystem
			if params.RootFS.Path != "" {
				cmd.AddParam("root", params.RootFS.Path)
				if params.RootFS.Options != "" {
					cmd.AddParam("rootflags", params.RootFS.Options)
				}
			}
		}
	}

	// Add cloud-init server configuration if specified
	if params.CloudInit != nil && params.CloudInit.URL != "" {
		if params.CloudInit.Version != "" {
			cmd.AddParam("ds", fmt.Sprintf("nocloud-net;s=%s;v=%s", params.CloudInit.URL, params.CloudInit.Version))
		} else {
			cmd.AddParam("ds", fmt.Sprintf("nocloud-net;s=%s", params.CloudInit.URL))
		}
	}

	script.WriteString(kernelCmd + " " + cmd.String() + " || goto boot_retry\n")

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
