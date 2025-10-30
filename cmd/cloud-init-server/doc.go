// Package main implements the OpenCHAMI cloud-init server.
//
// It serves nocloud-net compatible endpoints (meta-data, user-data, vendor-data)
// for cluster nodes using inventory from the System Management Database (SMD),
// plus cluster defaults and group overrides. The server can optionally restrict
// access to these endpoints through a WireGuard interface.
//
// WireGuard support can be provided by either the kernel module or a userspace
// implementation (via wireguard-go). Engine selection is controlled by flags
// or environment variables:
//   - --wireguard-server / WIREGUARD_SERVER: server WG IP/CIDR (e.g. 100.97.0.1/16)
//   - --wireguard-only  / WIREGUARD_ONLY: only allow access via the WG subnet/interface
//   - --wg-engine       / WG_ENGINE: kernel|userspace|auto (default auto)
//   - --fips-mode       / FIPS_MODE: when true and engine=auto, select userspace to
//     keep the kernel FIPS-compliant.
//
// Other notable flags/envs include SMD configuration, JWKS for secure routes,
// and storage backend selection. See README.md for full details.
package main
