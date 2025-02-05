# OpenCHAMI Cloud-Init Server

## Summary of Repo
The **OpenCHAMI cloud-init server** retrieves detailed inventory information from SMD and uses it to create cloud-init payloads customized for each node in an OpenCHAMI cluster.

## Table of Contents
1. [About / Introduction](#about--introduction)
2. [Overview](#overview)
   - [Unprotected Data](#unprotected-data)
     - [Setup with `user-data`](#setup-with-user-data)
     - [Generating Custom Metadata with Groups](#generating-custom-metadata-with-groups)
   - [JWT-Protected Data](#jwt-protected-data)
3. [Build / Install](#build--install)
4. [Testing](#testing)
5. [Running](#running)
6. [More Reading](#more-reading)

---

## About / Introduction
OpenCHAMI utilizes the **cloud-init** platform for post-boot configuration. As part of a quickstart Docker Compose setup, a custom cloud-init server container is included but must be populated before use.

> **Note**  
> This guide assumes the cloud-init server is accessible at `foobar.openchami.cluster`. If your setup differs, adapt the URLs accordingly.

---

## Overview

### Unprotected Data

#### Setup with `user-data`
User data can be injected into the cloud-init payload by making a `PUT` request to the `/cloud-init/{id}/user-data` endpoint. The request body should be JSON-formatted and can contain arbitrary data for your needs.

For example:
```bash
curl 'https://foobar.openchami.cluster/cloud-init/test/user-data' \
    -X PUT \
    -d '{
          "write_files": [
            {
              "content": "hello world",
              "path": "/etc/hello"
            }
          ]
        }'
```

This creates a payload similar to:
```json
{
  "name": "IDENTIFIER",
  "cloud-init": {
    "userdata": {
      "write_files": [
        {"content": "hello world", "path": "/etc/hello"}
      ]
    },
    "metadata": { ... }
  }
}
```

- `IDENTIFIER` can be:
  - A node MAC address
  - A node xname
  - An SMD group name

> **Note**  
> Data added via `PUT` only goes into the `userdata` section. There is no direct way to add data to the `metadata` section this way.

#### Generating Custom Metadata with Groups
The cloud-init server provides a **group API** to inject custom data into `metadata` when retrieving node data. It does this by looking up group labels from SMD and matching them with data submitted to the API.

1. Add group data to the cloud-init server:
   ```bash
   curl -k http://127.0.0.1:27780/cloud-init/groups/install-nginx \
        -d@install-nginx.json
   ```
2. `install-nginx.json` might look like this:
   ```json
   {
     "tier": "frontend",
     "application": "nginx"
   }
   ```
3. When a node in the `install-nginx` group fetches its data, the extra keys appear under `metadata.groups`:
   ```bash
   curl -k http://127.0.0.1:27780/cloud-init/x3000c1s7b53
   ```
   ```json
   {
     "name": "x3000c1s7b53",
     "cloud-init": {
       "userdata": { ... },
       "vendordata": { ... },
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

> **Important**  
> The specified `IDENTIFIER` must exist in SMD for the data injection to work.

#### Usage

Data is retrieved via HTTP GET requests to the `meta-data`, `user-data`, and `vendor-data` endpoints.

For example, one could download all cloud-init data for a node/group via `curl 'https://foobar.openchami.cluster/cloud-init/<IDENTIFIER>/{meta-data,user-data,vendor-data}'`.

When retrieving data, `IDENTIFIER` can also be omitted entirely (e.g. `https://foobar.openchami.cluster/cloud-init/user-data`). In this case, the cloud-init server will attempt to look up the relevant xname based on the request's source IP address.

Thus, the intended use case is to set nodes' cloud-init datasource URLs to `https://foobar.openchami.cluster/cloud-init/`, from which the cloud-init client will load its configuration data. Note that in this case, no `IDENTIFIER` is provided, so IP-based autodetection will be performed.

### JWT-Protected Data

#### Setup
Another set of endpoints at `/cloud-init-secure/` requires a **valid bearer token (JWT)** to access or store data. For example, using `curl`, you must include:

```bash
curl -k \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -X PUT \
  -d@secure-data.json \
  https://foobar.openchami.cluster/cloud-init-secure/test/user-data
```

This parallels the unprotected data workflow but adds token-based authorization.

#### Usage
When retrieving data for a node, you must include the JWT:

```bash
curl 'https://foobar.openchami.cluster/cloud-init-secure/IDENTIFIER/{meta-data,user-data,vendor-data}' \
  --create-dirs --output '/PATH/TO/DATA-DIR/#1' \
  --header "Authorization: Bearer $ACCESS_TOKEN"
```
Then, point cloud-init (on the node) to `file:///PATH/TO/DATA-DIR/` as its datasource URL.

> **Note**  
> Distributing or managing these tokens is out of scope here; see the [OpenCHAMI TPM-manager service](https://github.com/OpenCHAMI/TPM-manager) for a potential solution.

---

## Build / Install

This project uses **[GoReleaser](https://goreleaser.com/)** for building and releasing, embedding additional metadata such as commit info, build time, and version. Below is a brief overview for local builds.

### Environment Variables
To include detailed metadata in your builds, set the following:

- **GIT_STATE**: `clean` if your repo is clean, `dirty` if uncommitted changes exist  
- **BUILD_HOST**: Hostname of the build machine  
- **GO_VERSION**: Version of Go used (for consistent versioning info)  
- **BUILD_USER**: Username of the person/system performing the build  

```bash
export GIT_STATE=$(if git diff-index --quiet HEAD --; then echo 'clean'; else echo 'dirty'; fi)
export BUILD_HOST=$(hostname)
export GO_VERSION=$(go version | awk '{print $3}')
export BUILD_USER=$(whoami)
```

### Building Locally with GoReleaser
1. [Install GoReleaser](https://goreleaser.com/install/) following their documentation.  
2. Run in snapshot mode to build locally without releasing:

   ```bash
   goreleaser release --snapshot --clean
   ```
3. Check the `dist/` directory for compiled binaries, which will include the injected metadata.

> **Note**  
> If you encounter errors, ensure your GoReleaser version matches the one used in the [Release Action](.github/workflows/Release.yml).

---

## Testing
Currently, there are no specific automated tests in this repository (or details have not been provided). However, you can verify functionality by:

1. Uploading `user-data` or group configurations via `curl`  
2. Fetching them back with `curl https://foobar.openchami.cluster/cloud-init/<IDENTIFIER>/{meta-data,user-data,vendor-data}`  
3. Checking the returned JSON to confirm your data appears where expected  

You can also incorporate additional tests or integration checks as needed for your environment.

---

## Running
- A **custom cloud-init server container** is included with the Docker Compose setup.  
- Once the container is running, you can access endpoints at `https://<HOSTNAME>/cloud-init/` or `https://<HOSTNAME>/cloud-init-secure/`, where `<HOSTNAME>` is typically `foobar.openchami.cluster`.

**Example (unprotected flow)**:
1. Run the Docker Compose setup:
   ```bash
   docker-compose up -d
   ```
2. Make a request to add user data:
   ```bash
   curl -X PUT \
     'https://foobar.openchami.cluster/cloud-init/test/user-data' \
     -d '{
           "write_files": [
             {
               "content": "hello world",
               "path": "/etc/hello"
             }
           ]
         }'
   ```
3. Retrieve and inspect data:
   ```bash
   curl 'https://foobar.openchami.cluster/cloud-init/test/meta-data'
   curl 'https://foobar.openchami.cluster/cloud-init/test/user-data'
   ```
**Example (JWT-protected flow)**:
1. Obtain (or generate) a valid `ACCESS_TOKEN`.  
2. Send or retrieve data via the `/cloud-init-secure/` endpoints with the bearer token:
   ```bash
   curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     'https://foobar.openchami.cluster/cloud-init-secure/test/user-data'
   ```

---

## More Reading

- [Official cloud-init documentation](https://cloud-init.io/)  
- [OpenCHAMI TPM-manager service](https://github.com/OpenCHAMI/TPM-manager)  
- [GoReleaser Documentation](https://goreleaser.com/)  
- [SMD Documentation (if available internally)](#)  

---
