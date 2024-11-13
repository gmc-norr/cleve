# cleve &mdash; sequencing run database

![go build and test](https://github.com/gmc-norr/cleve/actions/workflows/go.yaml/badge.svg)
![golangci-lint](https://github.com/gmc-norr/cleve/actions/workflows/golangci-lint.yaml/badge.svg)

---

Cleve is a database mainly for sequencing data, with an accompanying CLI, API and web-based dashboard.
The goal of Cleve is to make management of sequencing runs with their associated samples and metadata effortless.

## Requirements

- A mongodb instance
- [Illumina InterOp](https://github.com/Illumina/interop) binaries (tested with [v1.3.1](https://github.com/Illumina/interop/releases/tag/v1.3.1))

## Installation

```bash
git clone https://github.com/gmc-norr/cleve
cd cleve
go generate ./...
go build -o ./bin/cleve ./cmd/cleve
```

The resulting binary is `./bin/cleve`.

## Configuration

Cleve looks for a yaml config file at startup.
The following locations are checked in this order:

- `/etc/cleve/config.yaml`
- `$HOME/.config/cleve/config.yaml`
- `$PWD/config.yaml`

The first config file that is found will be used, and the application will exit with an error if no config file is found.
The config can also be supplied with the `-c`/`--config` flag.

The config file has the following content:

```yaml
# Mongo database configuration
database:
  host: localhost   # mongodb host
  port: 27017       # mongodb port; 27017 is the default
  user: cleve       # database user
  password: secret  # password for user
  name: cleve       # database name

# If set, logs will be written to this file in addition to stdout
# If not set, logs will only be written to stdout
logfile: cleve.log

# Host and port where cleve will be served
host: 127.0.0.1
port: 8080
```

The only part that doesn't have decent defaults is the database.
If any required values are undefined the application will exit with an error.

## CLI

```
Interact with the sequencing database

Usage:
  cleve [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  db          Database management
  help        Help about any command
  key         API key management
  platform    Manage sequencing platforms
  run         Interact with sequencing runs
  samplesheet Interact with sample sheets
  serve       Serve the cleve api

Flags:
  -c, --config string   config file
  -h, --help            help for cleve
  -v, --version         version for cleve

Use "cleve [command] --help" for more information about a command.
```

Use the `--help` flag on the command line for complete documentation of all commands.

## Serving the dashboard and the API

In order to serve the API and the dashboard, run

```bash
cleve serve
```

In a production environment, it is highly recommended to set this up with a reverse proxy.

## Development

### Requirements

- tailwindcss: for web development

### pre-commit

This project makes use of pre-commit hooks.
The current setup requires [pre-commit](https://pre-commit.com/) and [golangci-lint](https://github.com/golangci/golangci-lint) to be installed.

## Where does the name come from?

The name cleve is a tribute to what many consider to be the first female librarian in Sweden, [Cecilia Cleve](https://en.wikipedia.org/wiki/Cecilia_Cleve).
