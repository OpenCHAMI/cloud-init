package wgtunnel

import (
	"fmt"
)

// Engine abstracts how a WireGuard interface is provisioned/cleaned up.
// Implementations can back the interface with a kernel module or a userspace process.
//
// EnsureInterface must make sure an interface with the given name exists and is ready
// to be configured with `wg` and `ip` commands (but should not set address/keys/bring up).
// CleanupInterface should remove/stop the interface/process.
// Both should be idempotent where practical.
//
// Note: Interface configuration (listen-port, private-key, peers, IP address, link up)
// is handled by InterfaceManager.
type Engine interface {
	EnsureInterface(name string) error
	CleanupInterface(name string) error
}

// EngineType represents desired engine option.
// Supported values: "kernel", "userspace", "auto"
type EngineType string

const (
	// EngineKernel selects the kernel-backed WireGuard engine.
	EngineKernel EngineType = "kernel"
	// EngineUserspace selects the userspace WireGuard engine (wireguard-go).
	EngineUserspace EngineType = "userspace"
	// EngineAuto selects an engine based on environment (userspace in FIPS mode, otherwise kernel).
	EngineAuto EngineType = "auto"
)

// SelectEngine creates an Engine based on inputs.
// If engineType is "auto", fipsMode=true forces userspace, otherwise kernel.
func SelectEngine(engineType EngineType, fipsMode bool) (Engine, error) {
	switch engineType {
	case EngineKernel:
		return NewKernelEngine(), nil
	case EngineUserspace:
		return NewUserspaceEngine(), nil
	case EngineAuto:
		if fipsMode {
			return NewUserspaceEngine(), nil
		}
		return NewKernelEngine(), nil
	default:
		return nil, fmt.Errorf("unknown engine type: %s", engineType)
	}
}
