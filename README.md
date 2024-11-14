# OpenCHAMI cloud-init Server

The OpenCHAMI cloud-init server retrieves detailed inventory information from SMD and uses it to create cloud-init payloads customized for each node in an OpenCHAMI cluster.

## cloud-init Server Setup

OpenCHAMI utilizes the cloud-init platform for post-boot configuration. A custom cloud-init server container is included with this quickstart Docker Compose setup, but must be populated prior to use.

The cloud-init server provides multiple API endpoints, described in the sections below. Choose the appropriate option for your needs, or combine them as appropriate.

> [!NOTE]
> This guide assumes that the cloud-init server is exposed as `foobar.openchami.cluster`.
> If your instance is named differently, adapt the examples accordingly.

### Unprotected Data

#### Setup with `user-data`

User data can be injected into the cloud-init payload by making a PUT request to the `/cloud-init/{id}/user-data` endpoint. The request should contain a JSON-formatted body and can contain any arbritrary data desired.

For example:

```bash
curl 'https://foobar.openchami.cluster/cloud-init/test/user-data' \
    -X PUT \
    -d '{
            "write_files": [{"content": "hello world", "path": "/etc/hello"}]
        }
    }}'
```

...which will create a payload that will look like the example below. Notice that the `id` gets injected into the payload as the `name` property.

```json
{
    "name": "IDENTIFIER", 
    "cloud-init": {
        "userdata": {
            "write_files": [{"content": "hello world", "path": "/etc/hello"}]
        },
        "metadata": {...},
    }
}
```

> [!NOTE]
> Data can only be added manually into the `userdata` section of the payload. There is no way to add data directly to the `metadata` section.

`IDENTIFIER` can be:

- A node MAC address
- A node xname
- An SMD group name

It may be easiest to add nodes to a group for testing, and upload a cloud-init configuration for that group to this server.

#### Generating Custom Metadata with Groups

The cloud-init server provides a group API to inject custom arbitrary data into the metadata section when requesting data with the specified `IDENTIFIER` mentioned above. The server will do a lookup for group labels from SMD and match the labels with the submitted data through the API.

For example, to add group data to the cloud-init server, we can make the following request:

```bash
curl -k http://127.0.0.1:27780/cloud-init/groups/install-nginx -d@install-nginx.json
```

The JSON data in `install-nginx.json`:

```json
{
    "tier": "frontend",
    "application": "nginx"
}
```

Now, when we fetch data for a node in the `install-nginx` group, this data will be injected into the cloud-init metadata. Additionally, we can view all groups stored in cloud-init by making a GET request to the `/cloud-init/groups` endpoint.

```bash
curl -k http://127.0.0.1:27780/cloud-init/x3000c1s7b53
```

And the response:

```json
{
  "name": "x3000c1s7b53",
  "cloud-init": {
        "userdata": {...},
        "vendordata": {...},
        "metadata": {
            "groups": {
                "install-nginx": {
                    "data": {
                        "application": "nginx",
                        "tier": "frontend"
                    }
                }
            }
        }
    }
}
```

> [!NOTE]
> The `IDENTIFIER` specified MUST be added to SMD for the data injection to work correctly.

#### Usage

Data is retrieved via HTTP GET requests to the `meta-data`, `user-data`, and `vendor-data` endpoints.

For example, one could download all cloud-init data for a node/group via `curl 'https://foobar.openchami.cluster/cloud-init/<IDENTIFIER>/{meta-data,user-data,vendor-data}'`.

When retrieving data, `IDENTIFIER` can also be omitted entirely (e.g. `https://foobar.openchami.cluster/cloud-init/user-data`). In this case, the cloud-init server will attempt to look up the relevant xname based on the request's source IP address.

Thus, the intended use case is to set nodes' cloud-init datasource URLs to `https://foobar.openchami.cluster/cloud-init/`, from which the cloud-init client will load its configuration data. Note that in this case, no `IDENTIFIER` is provided, so IP-based autodetection will be performed.

### JWT-Protected Data

#### Setup

The second endpoint, located at `/cloud-init-secure/`, restricts access to its cloud-init data behind a valid bearer token (i.e. a JWT).
Storing data into this endpoint requires a valid access token, which we assume is stored in `$ACCESS_TOKEN`.
The workflow described for unprotected data can be used, with the addition of the required authorization header, via e.g. `curl`'s `-H "Authorization: Bearer $ACCESS_TOKEN"`.

#### Usage

In order to access this protected data, nodes must also supply valid JWTs.
Distribution of these tokens is out-of-scope for this repository, but may be handled via the [OpenCHAMI TPM-manager service](https://github.com/OpenCHAMI/TPM-manager).

Once the JWT is known, it can be used to authenticate with the cloud-init server, via an invocation such as:

```bash
curl 'https://foobar.openchami.cluster/cloud-init-secure/<IDENTIFIER>/{meta-data,user-data,vendor-data}' \
    --create-dirs --output '/PATH/TO/DATA-DIR/#1' \
    --header "Authorization: Bearer $ACCESS_TOKEN"
```

cloud-init (i.e. on the node) can then be pointed at `file:///PATH/TO/DATA-DIR/` as its datasource URL.
