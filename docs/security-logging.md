<!-- SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC -->
<!-- SPDX-License-Identifier: MIT -->

# WireGuard startup diagnostics

Successful WireGuard startup logs the interface name, public key, listen port and
server address. It does not log the server private key at INFO or DEBUG level.
Key generation, temporary-file cleanup and interface configuration are unchanged.

This change requires no configuration or client migration. It does not introduce
JWT requirements or change API access, bootstrap identity or peer ownership.
Previously retained logs may already contain the private key; removing the log
field does not rotate that key or remove old log copies.
