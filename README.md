<div align="center">

# Snowflake.SnowCTL

_An opinionated Snowflake operations CLI for configuring connections, running ad‑hoc SQL, and automating workflows._

</div>

## Table of contents

1. [Overview](#overview)  
2. [Features](#features)  
3. [Quick start](#quick-start)  
4. [Configuration & secrets](#configuration--secrets)  
5. [CLI usage](#cli-usage)  
6. [Shell completions](#shell-completions)  
7. [Development workflow](#development-workflow)  
8. [Security notes](#security-notes)  
9. [Contributing](#contributing)

## Overview

`snowctl` is a Go‑based CLI that wraps common Snowflake administration tasks. It keeps a local inventory of named connections, securely tests those profiles, and executes SQL statements against the currently active connection. The root command lives in [`cmd/snowctl`](cmd/snowctl) and the shared packages under [`pkg/`](pkg) handle configuration, runtime state, Snowflake connectivity, and output formatting.

The CLI is intentionally scriptable: every command emits structured output (JSON by default, with YAML/CSV/TSV options) so that workflows can be automated in CI pipelines or shell scripts.

## Features

- **Connection lifecycle management**
  - `connection set` interactively gathers connection metadata and stores it under `~/.snowctl/config`.
  - `connection list`, `use`, `set-default`, and `remove` make it easy to inspect and rotate contexts.
  - `connection test` validates credentials (password or PAT) and can optionally mark the connection as current.
- **SQL execution**
  - `sql --query` runs statements against the active (or overridden) connection and returns normalized rows.
  - Output metadata always contains the connection name and statement, with rows serialized last.
- **Runtime controls**
  - Global `--connection` flag temporarily overrides the active context.
  - Global `--output` flag toggles between `json`, `yaml`, `csv`, and `tsv`.
- **Completions and metadata**
  - `completion` subcommand (and corresponding `make` targets) generate shell completion scripts.
  - `version` mirrors common CLIs (`kubectl`, `docker`) with `--output short|json`.
- **Well‑tested packages**
  - Unit tests cover configuration persistence, output normalization, connection workflows, and Snowflake mocking.

## Quick start

### Prerequisites

- Go **1.25.4** or newer (see [`go.mod`](go.mod)).
- A Snowflake account and credentials (username/password or PAT).

### Build & install

Clone the repo and build the CLI:

```bash
git clone https://github.com/Snowflake-Labs/Snowflake.SnowCTL.git
cd Snowflake.SnowCTL
go build -o snowctl ./cmd/snowctl
# or install into GOPATH/bin:
go install ./cmd/snowctl
```

### First connection

1. Run `./snowctl connection set MyConnection`.
2. Follow the prompts. Values default to any `SNOWFLAKE_*` environment variables present.
3. When prompted for the secret, enter the password or PAT. The CLI stores it only inside your `~/.snowctl/config` file (see [Security notes](#security-notes)).

You can then validate and activate the connection:

```bash
./snowctl connection test --set-current MyConnection
```

## Configuration & secrets

- Config location: `${HOME}/.snowctl/config`.  
  The file uses TOML and contains `currentContext`, `defaultContext`, and a `[contexts.<name>]` entry per connection.

```toml
[contexts.Analytics]
account    = "xy12345.us-east-1"
user       = "myuser"
role       = "SYSADMIN"
warehouse  = "REPORTING_WH"
database   = "FINANCE"
schema     = "PUBLIC"
authMethod = "password"
secret     = "••••"  # stored exactly as entered
```

- Secrets are no longer read from environment variables during runtime. Each connection stores its own password/PAT so that multiple profiles can coexist.
- Non-interactive automation can provide the secret via `snowctl connection set --secret "$SECRET"`; when `--no-prompt` is used, all required flags plus `--secret` must be supplied.
- The `connection test` and `sql` commands fail fast when a profile lacks a stored credential, prompting you to rerun `connection set`.

## CLI usage

### Built-in help

```
Snowflake DevOps CLI (v0.0.1)

Usage:
  snowctl <command> [flags]

Commands:
  completion   Generate shell completion scripts
  connection   Manage Snowflake connections
  sql          Execute SQL against the active connection

Flags:
  -h, --help         Show help
  -v, --version      Show version
  -c, --connection   Use a connection
  -o, --output       Output format
```

### Global flags

| Flag | Description |
|------|-------------|
| `-c, --connection NAME` | Temporarily override the active connection for the current command. |
| `-o, --output FORMAT`   | Output format (`json`, `yaml`, `csv`, or `tsv`). Defaults to `json`. |

### Connection management

| Command | Description |
|---------|-------------|
| `snowctl connection set [NAME]` | Create or update a connection (interactive by default). Supports `--auth-method password|pat`, `--secret`, `--make-current`, and `--no-prompt`. |
| `snowctl connection list` | Display all connections with `isCurrent`/`isDefault` indicators. |
| `snowctl connection use NAME` | Switch the current connection for subsequent commands. |
| `snowctl connection set-default NAME` | Change the default connection used when no current override exists. |
| `snowctl connection remove NAME` | Delete a stored connection. |
| `snowctl connection test [NAME]` | Validate connectivity, optionally selecting from a prompt when NAME is omitted. `--set-current` flips the connection on success. |

### Running SQL

```bash
./snowctl sql --query "select current_schema()"
```

Output (JSON by default) is structured as:

```json
{
  "connection": "Analytics",
  "statement": "select current_schema()",
  "rows": [
    { "CURRENT_SCHEMA()": "PUBLIC" }
  ]
}
```

When `--output csv`/`tsv` is used, metadata (connection + statement) precedes the tabular rows to keep scripts machine friendly.

### Version & completion

- `snowctl version` prints the build version (short or JSON).
- `snowctl completion <shell>` emits completion scripts. The provided [`Makefile`](Makefile) has convenience targets (`make completion-zsh`, etc.) that write the files into `~/.snowctl/completions/`.

## Shell completions

Generate scripts manually:

```bash
./snowctl completion zsh > /usr/local/share/zsh/site-functions/_snowctl
./snowctl completion bash > /etc/bash_completion.d/snowctl
./snowctl completion fish > ~/.config/fish/completions/snowctl.fish
./snowctl completion powershell > snowctl.ps1
```

Or run `make completions` to emit all four variants into `~/.snowctl/completions/`.

## Development workflow

### Tests

```bash
go test ./...
```

The suite exercises:

- Configuration loading/migration.
- Runtime enforcement of current/default contexts.
- Connection workflows (set/list/use/test) with in-memory configs.
- SQL output ordering and CSV/TSV normalization.
- Snowflake connectivity via `sqlmock`.

### Formatting & linting

Go files should be formatted with `gofmt`. CI expectations:

```bash
gofmt -w <files>
go test ./...
```

### Building

```bash
go build -o snowctl ./cmd/snowctl
```

For release builds, inject version metadata:

```bash
go build -o snowctl \
  -ldflags "-X github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/build.Version=vX.Y.Z \
            -X github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/build.Commit=$(git rev-parse HEAD) \
            -X github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  ./cmd/snowctl
```

## Security notes

- Secrets are stored **unencrypted** inside `~/.snowctl/config`. The file is created with `0600` permissions; ensure your workstation honours that and restricts access to trusted administrators only.
- Always rotate PATs/passwords when personnel change, and rerun `snowctl connection set` to update the stored secret.
- `snowctl` never prints secrets, and tests rely on in-memory mocks so real credentials remain untouched.

## Contributing

1. Fork and clone the repo.
2. Create a topic branch from `main`.
3. Make your changes, keeping commits focused.
4. Run `gofmt` and `go test ./...`.
5. Open a PR describing the change and any testing performed.

Issues and feature requests are welcome through GitHub Issues. For security disclosures, please contact the maintainer privately before filing a public issue.
