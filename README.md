# Turso CLI

[![Go](https://github.com/chiselstrike/turso-cli/actions/workflows/go.yml/badge.svg)](https://github.com/chiselstrike/turso-cli/actions/workflows/go.yml)

This is the command line interface (CLI) to Turso.

## Installation

### Package Manager

#### [Homebrew](https://brew.sh) (macOS, Linux, WSL)

```bash
brew install chiselstrike/tap/turso
```

Also remember to configure `homebrew` [shell completions](https://docs.brew.sh/Shell-Completion) if you haven't already done so.

### Install Script

```bash
curl -sSfL https://get.tur.so/install.sh
```

### Building from Sources

```bash
cd cmd/turso && go install
```

## Usage

### Authentication

To authenticate with the service, run:

```bash
turso auth login
```

You are taken to a web page in your default browser to authenticate via GitHub.
After succesfully authenticated, `turs auth login` receives an access token that is stored on your settings file.

### Create database

To create a database, run:

```bash
turso db create
```

You can configure the database name with:

```bash
turso db create <database name>
```

### Start SQL shell

You can start an interactive SQL shell similar to `sqlite3` or `psql` with:

```bash
turso db shell <database name>
```

### Replicate database

First, list available regions and pick a region you want to replicate to:

```bash
turso db regions
```

Then, to replicate a database, run:

```bash
turso db replicate <database name> <region>
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

The `turso` program keeps settings in your local machine in the following base directory in `turso/settings.json` file:

| OS    | Config directory |
| ----- | -----------------|
| Linux | `$XDG_CONFIG_HOME` or `$HOME/.config` |
| macOS | `$HOME/Library/Application Support`  |
