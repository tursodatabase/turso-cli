# Turso CLI

[![Go](https://github.com/tursodatabase/turso-cli/actions/workflows/go.yml/badge.svg)](https://github.com/tursodatabase/turso-cli/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://github.com/tursodatabase/turso-cli/blob/main/LICENSE)

This is the command line interface (CLI) to Turso.

## Documentation

There is [reference documentation](https://docs.turso.tech/reference/turso-cli)
available.

For a guided walkthrough, follow the
[Turso CLI tutorial](https://docs.turso.tech/tutorials/get-started-turso-cli).

## Installation

### Package manager

#### [Homebrew](https://brew.sh) (macOS, Linux, WSL)

```bash
brew install tursodatabase/tap/turso
```

Also remember to configure `homebrew`
[shell completions](https://docs.brew.sh/Shell-Completion) if you haven't
already done so.

To upgrade an existing installation of the CLI, run:

```bash
brew upgrade turso
```

### Install script

```bash
curl -sSfL https://get.tur.so/install.sh | bash
```

### Go

```bash
go install github.com/tursodatabase/turso-cli/cmd/turso@latest
```

### Building from source

```bash
cd cmd/turso && go install
```

## Usage

### Authentication

If you are a new user, you can sign up with:

```bash
turso auth signup
```

If you already have an account, authenticate with:

```bash
turso auth login
```

You are taken to a web page in your default browser to authenticate via GitHub.
After successfully authenticated, `turso auth login` receives an access token
that is stored on your settings file.

### Create database

To create a database with a generated name, run:

```bash
turso db create
```

Or you can specify the name of the database with:

```bash
turso db create <database name>
```

### Start SQL shell

You can start an interactive SQL shell similar to `sqlite3` with:

```bash
turso db shell <database name>
```

### Run SQL command without interactive shell

You can also run a command without using the interactive SQL shell:

```bash
turso db shell <database name> <sql query>
```

### Replicate database

First, list available locations and pick a location you want to replicate to:

```bash
turso db locations
```

Then, to replicate a database, run:

```bash
turso db replicate <database name> <location>
```

### List databases

To list your databases, run:

```bash
turso db list
```

### Delete database

```bash
turso db destroy <database name>
```

## Settings

The `turso` program keeps settings in your local machine in the following base
directory in `turso/settings.json` file:

| OS    | Config directory                          |
| ----- | ----------------------------------------- |
| Linux | `$XDG_CONFIG_HOME` or `$HOME/.config`     |
| macOS | `$HOME/Library/Application Support/turso` |
