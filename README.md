# Cleve &mdash; sequencing run database

![go build and test](https://github.com/gmc-norr/cleve/actions/workflows/go.yaml/badge.svg)
![golangci-lint](https://github.com/gmc-norr/cleve/actions/workflows/golangci-lint.yaml/badge.svg)

---

Cleve is a database mainly for sequencing data, with an accompanying CLI, API and web-based dashboard.
The goal of Cleve is to make management of sequencing runs with their associated samples and metadata effortless.
This is achieved by storing metadata for the sequencing runs, including QC data.

Interactive visualisations of various parameters are included in the dashboard, albeit rudimentary at the moment.
In essence, this provides some of the same features as Illumina Sequencing Analysis Viewer, but the difference is that it is much faster, and there is also the possibility to visualise QC parameters across runs.
These visualisations provide a quick way of visually evaluate the state of a sequencing run.

## Requirements

- A mongodb instance

## Docker compose

The simplest way to spin up a test environment of Cleve is to use docker compose.

```
docker compose up -d --build
```

This will spin up three containers: one for Cleve, one for MongoDB, and one for Keycloak (see [Authentication](#Authentication).
Cleve will by default be served on `http://127.0.0.1:8080`, and Keycloak will be served on `http://127.0.0.1:8088`.

When spinning up the environment, a realm, client, and two users are added to Keycloak.
Only one of these users are also added to Cleve do demonstrate how it looks when a user is able to authenticate with Keycloak, but is not authorized by Cleve.

The users are:

Name     | Username | Email                  | Password   | Authorized
---------|----------|------------------------|------------|-------------------
John Doe | jodo01   | `john.doe@example.com` | `password` | Yes
Jane Doe | jado01   | `jane.doe@example.com` | `password` | No

In order to add new users, they have to be added to both Keycloak and Cleve.
You can access the Keycloak admin interface on `http://127.0.0.1:8088` with username `admin` and password `admin`.

## Installation

The easiest way to build Cleve is to use make:

```bash
git clone https://github.com/gmc-norr/cleve
cd cleve
make
```

The manual steps required are:

```bash
go generate ./...
go build -o ./bin/cleve ./cmd/cleve
```

The resulting binary is `./bin/cleve`.

## Configuration

Cleve looks for a yaml config file at startup.
The following locations are checked in this order:

- `$PWD/config.yaml`
- `$XDG_CONFIG_HOME/cleve/config.yaml`
- `$HOME/.config/cleve/config.yaml`
- `/etc/cleve/config.yaml`

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

# Authentication
auth:
  provider: none
```

The only part that doesn't have decent defaults is the database.
If any required values are undefined the application will exit with an error.

## CLI

```
❯ cleve --help
Interact with the sequencing database

Usage:
  cleve [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  db          Database management
  help        Help about any command
  key         API key management
  panel       Manage in-silico gene panels
  platform    Manage sequencing platforms
  run         Interact with sequencing runs
  samplesheet Interact with sample sheets
  serve       Serve the cleve api
  user        Manage users

Flags:
  -c, --config string            config file
  -h, --help                     help for cleve
  -l, --loglevel string          Logging verbosity (case-insensitive: DEBUG, INFO, WARN, ERROR) (default "WARN")
  -v, --version                  version for cleve
      --webhook-api-key string   API key for the webhook service ("<header-key>=<api-key>")
      --webhook-url string       URL to send webhook messages to

Use "cleve [command] --help" for more information about a command.
```

Use the `--help` flag on the command line for complete documentation of all commands.

## Serving the dashboard and the API

In order to serve the API and the dashboard, run

```bash
cleve serve
```

In a production environment, it is highly recommended to set this up with a reverse proxy.

Once everything is up and running, the API documentation can be found at `<cleve-host>:<cleve-port>/api`.

## Authentication

There are three options for authentication for the web interface: no authentication (default, use with care), internal authentication, and Keycloak authentication.

### No authentication

With no authentication, a user will never be asked to authenticate, and all content will be available to them.
This should not be used in a production setting. This is the default setting.

#### Configuration

```yaml
auth:
  provider: none
```

### Internal authentication

Internal authentication stores user credentials in the Cleve database, and Cleve handles the authentication.
Users can be managed with the CLI command `cleve user`:


```
❯ cleve user --help
Manage users

Usage:
  cleve user [command]

Available Commands:
  create      Create a new user
  delete      Delete a user
  list        List users
  update      Update user details

Flags:
  -h, --help   help for user

Global Flags:
  -c, --config string            config file
  -l, --loglevel string          Logging verbosity (case-insensitive: DEBUG, INFO, WARN, ERROR) (default "WARN")
      --webhook-api-key string   API key for the webhook service ("<header-key>=<api-key>")
      --webhook-url string       URL to send webhook messages to

Use "cleve user [command] --help" for more information about a command.
```

#### Configuration

```yaml
auth:
  provider: internal
```

### Keycloak authentication

Keycloak authentication delegates the authentication to [Keycloak](https://www.keycloak.org/).
For this to work, a Keycloak server needs to be avaliable and configured with a realm and client that Cleve can use.
Keycloak only handles the authentication side of things.
Authorization is handled by Cleve, so users that should have access should also be added to the Cleve database using the CLI (see previous section).

There is a Keycloak service set up in the Docker compose environment in this repo, and this can be used to get familiar with Keycloak.
It should require no extra setup, but work out of the box.

#### Configuration

```yaml
auth:
  provider: keycloak
  realm_url: https://keycloak.example.com/realms/cleve # Full Keycloak realm URL
  client: cleve # Keycloak client ID
  secret: supersecretkey # Client secret provided by Keycloak
```

## Outgoing webhooks

Cleve has the ability to send messages to a webhook endpoint.
In order of precedence, this can be configured in

- Command line parameters
- Environment variables
- The config file

The configuration consists of a URL and an optional API key/token if it is needed for authentication.
If only the URL is supplied, it is assumed that no authentication is needed.
The command line parameters are `--webhook-url` and `--webhook-api-key`, the environment variables are `CLEVE_WEBHOOK_URL` and `CLEVE_WEBHOOK_API_KEY`, and the config keys are `webhook_url` and `webhook_api_key`.
The URL should point to the exact endpoint that should be used, including protocol and port.
If an API key is needed, this should be on the format `<header-key>=<header-value>` where `<header-key>` is the HTTP header that is expected by the receiving endpoint, and the `<header-value>` is the API key.

Messages are sent to this endpoint whenever the state of a run or an analysis is updated.
The message is send with `Content-Type: application/json` using HTTP POST, and the JSON body is a single object with the following keys:

- `unit`: the entity represented in the message, either `"run"` or `"analysis"`
- `id`: the ID of the run or analysis
- `platform`: if unit is `"run"`, this is the sequencing platform; if unit is `"analysis"`, this is the analysis software
- `message`: free text message
- `message_type`: the only message type currently implemented is `"state_update"`.
- `state`: the most recent state of the run or analysis
- `path`: absolute path to the run or analysis directory
- `time`: date and time the message was generated (not the time when the status was changed)

## Development

### Requirements

In addition to the general requirments, the following are recommended in a development environment:

- [tailwindcss 4](https://tailwindcss.com/): for web development (the [standalone CLI](https://tailwindcss.com/blog/standalone-cli) is recommended)
- [pre-commit](https://pre-commit.com): if pre-commit is used, [golangci-lint](https://github.com/golangci/golangci-lint) is also a requirement

### Testing

Unit tests can be run with `make test` or `go test ./...`.

## Where does the name come from?

The name cleve is a tribute to what many consider to be the first female librarian in Sweden, [Cecilia Cleve](https://en.wikipedia.org/wiki/Cecilia_Cleve).
