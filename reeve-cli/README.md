# Reeve CI / CD - Command Line Tools

## Installing

```sh
go install github.com/reeveci/reeve-cli/reeve@latest

reeve --help
```

## Configuration

Configuration options can be set using either environment variables (with prefix `REEVE_CLI_`), CLI flags or a TOML file.

The configuration file `.reevecli` is stored in a directory `reeve` in the user's configuration directory (`$HOME/.config/reeve` on linux) and can be otherwise specified with the `REEVE_CLI_CONFIG` environment variable.

The configuration file can be managed using the `config` command.

```sh
reeve config set url https://reeve-server:9080
REEVE_CLI_SECRET='supersecret'

reeve ask --list
```
