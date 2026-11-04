# mcpdiff

**Catch breaking changes in your MCP server before your users' agents do.**

`mcpdiff` compares two snapshots of an [MCP](https://modelcontextprotocol.io) server's `tools/list` response and reports what changed: removed tools, removed or newly required parameters, type changes, dropped enum values, and description edits that can change how agents pick tools. It exits non-zero on breaking changes, so it drops straight into CI. Zero dependencies, single Go binary.

## Install

```sh
go install github.com/celaldrg/mcpdiff@latest
```

Or grab a binary from [Releases](https://github.com/celaldrg/mcpdiff/releases).

## Usage

```sh
mcpdiff [-json] [-fail-on breaking|warning|never] OLD.json NEW.json
```

Snapshots can be a raw JSON-RPC response (`{"result":{"tools":[...]}}`), a `{"tools":[...]}` object, or a bare array of tools.

## Example

```text
$ mcpdiff testdata/old.json testdata/new.json
BREAKING  fetch_page           tool removed
WARN      search               description changed (agents may pick tools differently)
BREAKING  search               limit.type changed integer -> string
BREAKING  search               mode.enum value deep removed
BREAKING  search               new required param index
info      search               new optional param lang
info      summarize            tool added

4 breaking, 1 warnings, 2 info
```

## In CI

Commit a snapshot of your server's `tools/list` output (for example `mcp-tools.json`), regenerate it in CI, and run:

```sh
mcpdiff mcp-tools.json mcp-tools.new.json
```

Use `-json` for machine-readable output and `-fail-on warning` to also fail on description changes.

## What counts as breaking

| Change | Level |
|---|---|
| Tool removed | breaking |
| Parameter removed | breaking |
| Parameter type changed | breaking |
| Enum value removed | breaking |
| New required parameter / parameter became required | breaking |
| Tool description changed | warning |
| Tool added / new optional parameter | info |

Nested object properties are checked recursively.

## License

MIT © Celalettin Durğaç
