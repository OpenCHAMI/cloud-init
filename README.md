# OpenCHAMI cloud-init Server

The OpenCHAMI cloud-init server, which retrieves detailed inventory information from SMD and uses it to create cloud-init payloads customized for each node in an OpenCHAMI cluster.

## cloud-init Server Setup

OpenCHAMI utilizes the cloud-init platform for post-boot configuration.
A custom cloud-init server container is included with this quickstart Docker Compose setup, but must be populated prior to use.

The cloud-init server provides two API endpoints, described in the sections below.
Choose the appropriate option for your needs, or combine them as appropriate.

> [!NOTE]
> This guide assumes that the cloud-init server is exposed as `foobar.openchami.cluster`.
> If your instance is named differently, adapt the examples accordingly.

### Unprotected Data

#### Setup
The first endpoint, located at `/cloud-init/`, permits access to all stored data (and should therefore not contain configuration secrets).
Storing data into this endpoint is accomplished via HTTP POST requests containing JSON-formatted cloud-init configuration details.
For example:
```bash
curl 'https://foobar.openchami.cluster/cloud-init/' \
    -X POST \
    -d '{"name": "IDENTIFIER", "cloud-init": {
        "userdata": {
            "write_files": [{"content": "hello world", "path": "/etc/hello"}]
        },
        "metadata": {...},
        "vendordata": {...}
    }}'
```
`IDENTIFIER` can be:
- A node MAC address
- A node xname
- An SMD group name
It may be easiest to add nodes to a group for testing, and upload a cloud-init configuration for that group to this server.

#### Usage
Data is retrieved via HTTP GET requests to the `meta-data`, `user-data`, and `vendor-data` endpoints.
For example, one could download all cloud-init data for a node/group via `curl 'https://foobar.openchami.cluster/cloud-init/<IDENTIFIER>/{meta-data,user-data,vendor-data}'`.

When retrieving data, `IDENTIFIER` can also be omitted entirely (e.g. `https://foobar.openchami.cluster/cloud-init/user-data`).
In this case, the cloud-init server will attempt to look up the relevant xname based on the request's source IP address.

Thus, the intended use case is to set nodes' cloud-init datasource URLs to `https://foobar.openchami.cluster/cloud-init/`, from which the cloud-init client will load its configuration data.
Note that in this case, no `IDENTIFIER` is provided, so IP-based autodetection will be performed.

### JWT-Protected Data

#### Setup
The second endpoint, located at `/cloud-init-secure/`, restricts access to its cloud-init data behind a valid bearer token (i.e. a JWT).
Storing data into this endpoint requires a valid access token, which we assume is stored in `$ACCESS_TOKEN`.
The workflow described for unprotected data can be used, with the addition of the required authorization header, via e.g. `curl`'s `-H "Authorization: Bearer $ACCESS_TOKEN"`.

#### Usage
In order to access this protected data, nodes must also supply valid JWTs.
These may be distributed to nodes at boot time by including [`tpm-manager.yml`](tpm-manager.yml) in the `docker compose` command provided above.
(It should be provided last, since it supplements service definitions from other files.)

For nodes without hardware TPMs, the TPM manager will drop a JWT into `/var/run/cloud-init-jwt`.
The token can be retrieved and included with a request to the cloud-init server, using an invocation such as:
```bash
curl 'https://foobar.openchami.cluster/cloud-init-secure/<IDENTIFIER>/{meta-data,user-data,vendor-data}' \
    --create-dirs --output '/PATH/TO/DATA-DIR/#1' \
    --header "Authorization: Bearer $(</var/run/cloud-init-jwt)"
```
cloud-init (i.e. on the node) can then be pointed at `file:///PATH/TO/DATA-DIR/` as its datasource URL.

For nodes *with* hardware TPMs, the token will be dropped into the TPM's secure storage with `ownerread|ownerwrite` scope, and can be retrieved using the `tpm2_nvread` command.
