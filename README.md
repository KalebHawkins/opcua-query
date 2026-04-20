# opcua-query

Small Go CLI for testing OPC UA connections, browsing node paths, previewing SiteWise-compatible node filters, and copying the resulting filter payload for AWS IoT SiteWise.

Repository: `github.com/KalebHawkins/opcua-query`

## Features

- Connect to an OPC UA server with `hostname:port` or `opc.tcp://hostname:port`
- Browse from the OPC UA Objects folder or a supplied start node id
- Match discovered browse paths against AWS IoT SiteWise-style filters using `/`, `*`, and `**`
- Read live values for matched variable nodes
- Render a colorized report in the terminal
- Copy the generated SiteWise filter JSON or raw root path to the clipboard

## Installation

Install the latest version directly with Go:

```powershell
go install github.com/KalebHawkins/opcua-query@latest
```

Build from source locally:

```powershell
git clone https://github.com/KalebHawkins/opcua-query.git
cd opcua-query
go build ./...
```

After `go install`, use the `opcua-query` binary from your Go bin directory.

## Usage

```powershell
opcua-query browse --server localhost:4840 --filter "/**/PLC*"
opcua-query browse --server ignition.local:62541 --filter "/Plant/Area 1/**" --copy
opcua-query browse --server 10.0.0.25:53530 --username operator --password secret --filter "/Line 2/Counter*"
```

Run from source during development:

```powershell
go run . browse --server opc.tcp://ignition.local:62541 --filter "/Tag Providers/MyProvider/**"
```

## Options

```text
--server        OPC UA server in hostname:port or opc.tcp://hostname:port format
--filter        SiteWise-compatible rootPath expression
--start-node    Optional OPC UA node id such as ns=0;i=85
--max-depth     Maximum browse depth
--max-nodes     Maximum number of nodes to inspect
--read-values   Read values for matched variable nodes
--copy          Copy the generated filter output to the clipboard
--copy-format   Clipboard format: json or path
--timeout       Overall connect and browse timeout
```

## Config file

`opcua-query.yaml` is optional. Flags override config values.

```yaml
server: ignition.local:62541
username: operator
password: secret
timeout: 2m
filter: "/**/PLC*"
start_node: "ns=0;i=85"
max_depth: 8
max_nodes: 1000
read_values: true
copy: false
copy_format: json
```

Environment variables use the `OPCUA_QUERY_` prefix. For example, `OPCUA_QUERY_SERVER`, `OPCUA_QUERY_TIMEOUT`, and `OPCUA_QUERY_FILTER`.

## Development

Run the local verification steps before pushing:

```powershell
go test ./...
go vet ./...
go build ./...
```

## SiteWise filters

The generated filter payload follows the SiteWise OPC UA source model:

```json
[
  {
    "action": "INCLUDE",
    "definition": {
      "type": "OpcUaRootPath",
      "rootPath": "/**/PLC*"
    }
  }
]
```

## CI

GitHub Actions runs formatting checks, tests, vetting, and builds on pushes and pull requests. The workflow is defined in `.github/workflows/ci.yml`.

