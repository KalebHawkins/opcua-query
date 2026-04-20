# opcua-query

Small Go CLI for testing OPC UA connections, browsing node paths, previewing SiteWise-compatible node filters, and copying the resulting filter payload for AWS IoT SiteWise.

Repository: `github.com/KalebHawkins/opcua-query`

## Features

- Connect to an OPC UA server with `hostname:port` or `opc.tcp://hostname:port`
- Browse from the OPC UA Objects folder or a supplied start node id
- List the immediate child nodes for a specific browse path or start node
- Match discovered browse paths against AWS IoT SiteWise-style filters using `/`, `*`, and `**`
- Explore the OPC UA hierarchy as a tree when you do not know the path yet
- Search nodes by name, display name, path fragment, or node id
- Read live values for matched variable nodes
- Subscribe to matched variable nodes and stream live value changes until stopped
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
opcua-query ls --server localhost:4840 --path "/Plant/Area 1"
opcua-query tree --server localhost:4840 --max-depth 3
opcua-query find --server localhost:4840 --name Counter
opcua-query browse --server ignition.local:62541 --filter "/Plant/Area 1/**" --copy
opcua-query browse --server 10.0.0.25:53530 --username operator --password secret --filter "/Line 2/Counter*"
opcua-query watch --server ignition.local:62541 --filter "/Plant/Area 1/Line 2/**" --interval 500ms
```

Run from source during development:

```powershell
go run . browse --server opc.tcp://ignition.local:62541 --filter "/Tag Providers/MyProvider/**"
go run . ls --server opc.tcp://ignition.local:62541 --path "/Tag Providers/MyProvider"
go run . tree --server opc.tcp://ignition.local:62541 --max-depth 2
go run . find --server opc.tcp://ignition.local:62541 --name Counter
go run . watch --server opc.tcp://ignition.local:62541 --filter "/Tag Providers/MyProvider/Counter*"
```

## Options

```text
--server        OPC UA server in hostname:port or opc.tcp://hostname:port format
--filter        SiteWise-compatible rootPath expression
--name          Search query for find mode
--path          Exact browse path to resolve for ls mode
--start-node    Optional OPC UA node id such as ns=0;i=85
--max-depth     Maximum browse depth
--max-nodes     Maximum number of nodes to inspect
--read-values   Read values for matched variable nodes
--copy          Copy the generated filter output to the clipboard
--copy-format   Clipboard format: json or path
--timeout       Overall connect and browse timeout
--interval      Subscription publishing interval for watch mode
```

## Watch mode

Use `watch` when you want a true OPC UA subscription instead of a snapshot read. The command discovers variable nodes that match the supplied filter, subscribes to them, and streams value changes until you press `Ctrl+C`.

```powershell
opcua-query watch --server localhost:4840 --filter "/**/Counter*"
opcua-query watch --server ignition.local:62541 --filter "/Plant/Area 1/**" --interval 250ms
```

When you stop the command with `Ctrl+C`, the CLI cancels the subscription and closes the OPC UA session cleanly before exiting.

## Tree mode

Use `tree` when you need to explore the hierarchy before you know the exact SiteWise-style path.

```powershell
opcua-query tree --server localhost:4840
opcua-query tree --server ignition.local:62541 --max-depth 3
opcua-query tree --server ignition.local:62541 --start-node "ns=0;i=85" --read-values
```

## ls mode

Use `ls` when you already know the parent location and only want the immediate child nodes, without recursively rendering the full tree.

```powershell
opcua-query ls --server localhost:4840
opcua-query ls --server ignition.local:62541 --path "/Plant/Area 1"
opcua-query ls --server ignition.local:62541 --start-node "ns=0;i=85" --read-values
```

## Find mode

Use `find` when the address space is too large to scan manually. It searches browse names, display names, paths, and node ids.

```powershell
opcua-query find --server localhost:4840 --name Counter
opcua-query find --server ignition.local:62541 --name "Area 1" --max-depth 5
opcua-query find --server ignition.local:62541 --name ns=2;s=Counter01
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
watch_interval: 1s
ls_path: "/Plant/Area 1"
ls_read_values: false
tree_max_depth: 4
tree_read_values: false
find_name: Counter
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

