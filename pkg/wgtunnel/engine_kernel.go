package wgtunnel

import (
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// kernelEngine provisions a kernel-backed WireGuard interface.

type kernelEngine struct{}

// NewKernelEngine returns a kernel-backed WireGuard engine.
func NewKernelEngine() Engine { return &kernelEngine{} }

func (e *kernelEngine) EnsureInterface(name string) error {
	cmd := exec.Command("ip", "link", "add", "dev", name, "type", "wireguard")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore if already exists
		if !strings.Contains(err.Error(), "File exists") {
			log.Warn().Str("output", string(out)).Msgf("Failed to create interface %s: %v", name, err)
		}
	}
	return nil
}

func (e *kernelEngine) CleanupInterface(name string) error {
	cmd := exec.Command("ip", "link", "delete", "dev", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Warn().Str("output", string(out)).Msgf("Failed to delete interface %s: %v", name, err)
	}
	return nil
}
