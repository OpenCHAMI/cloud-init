package wgtunnel

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// userspaceEngine provisions a userspace WireGuard interface via wireguard-go.

type userspaceEngine struct {
	mu    sync.Mutex
	procs map[string]*exec.Cmd
}

// NewUserspaceEngine returns a userspace-backed WireGuard engine using wireguard-go.
func NewUserspaceEngine() Engine {
	return &userspaceEngine{procs: make(map[string]*exec.Cmd)}
}

func (e *userspaceEngine) EnsureInterface(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.procs[name]; ok {
		return nil
	}
	cmd := exec.Command("wireguard-go", name)
	// Start in background and let it keep running.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start wireguard-go: %w", err)
	}
	e.procs[name] = cmd
	// Wait for UAPI readiness by polling wg show
	for i := 0; i < 30; i++ { // ~3s @ 100ms
		if out, err := exec.Command("wg", "show", name, "public-key").CombinedOutput(); err == nil && len(out) > 0 {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Warn().Msgf("wireguard-go interface %s did not become ready in time", name)
	return nil
}

func (e *userspaceEngine) CleanupInterface(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Try to delete link regardless
	_ = exec.Command("ip", "link", "delete", "dev", name).Run()
	if cmd, ok := e.procs[name]; ok {
		// Attempt graceful stop
		_ = cmd.Process.Kill()
		delete(e.procs, name)
	}
	return nil
}
