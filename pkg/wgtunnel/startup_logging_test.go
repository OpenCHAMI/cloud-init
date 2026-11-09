// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

package wgtunnel

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestStartServerDoesNotLogPrivateKey(t *testing.T) {
	const privateKey = "fixture-private-key-must-not-be-logged"
	const publicKey = "fixture-server-public-key"
	commands := t.TempDir()
	for name, script := range map[string]string{
		"ip": "#!/bin/sh\nexit 0\n",
		"wg": "#!/bin/sh\nif [ \"$1\" = show ]; then printf '%s\\n' '" + publicKey + "'; fi\n",
	} {
		if err := os.WriteFile(filepath.Join(commands, name), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// Only fake commands can run; this test never creates a host interface.
	t.Setenv("PATH", commands)
	keyFiles := t.TempDir()
	t.Setenv("TMPDIR", keyFiles)
	originalLogger := log.Logger
	t.Cleanup(func() { log.Logger = originalLogger })
	for _, level := range []zerolog.Level{zerolog.InfoLevel, zerolog.DebugLevel} {
		t.Run(level.String(), func(t *testing.T) {
			var output bytes.Buffer
			log.Logger = zerolog.New(&output).Level(level)
			manager := newTestInterfaceManager(t)
			manager.privateKey = privateKey
			if err := manager.StartServer(); err != nil {
				t.Fatal(err)
			}
			if manager.publicKey != publicKey || manager.privateKey != privateKey {
				t.Fatal("startup changed key configuration")
			}
			if strings.Contains(output.String(), privateKey) || strings.Contains(output.String(), "Private Key") {
				t.Error("WireGuard startup logged its private key")
			}
			if !strings.Contains(output.String(), "Wireguard Server Configured") || !strings.Contains(output.String(), publicKey) {
				t.Error("missing useful public startup diagnostics")
			}
			files, err := os.ReadDir(keyFiles)
			if err != nil || len(files) != 0 {
				t.Fatal("startup did not clean up the temporary private-key file")
			}
		})
	}
}
