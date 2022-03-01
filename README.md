# Nitric CLI
![GitHub Workflow Status (branch)](https://img.shields.io/github/workflow/status/nitrictech/cli/Tests/develop?label=tests)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/nitrictech/cli)
![GitHub all releases](https://img.shields.io/github/downloads/nitrictech/cli/total)


## Overview

Nitric provides a command-line interface (CLI) to assist with various tasks when building applications with [Nitric](https://nitric.io). To view a list of available commands, you can call the CLI without specifying any subcommands or arguments:

```bash
nitric
```

### Help and Documentation

Each command is self documented and provides a "help" interface describing the usage, arguments and options for the command. Use the `help` command to view the help information for any other command:

```bash
# Example displaying help for the `build` command
nitric help build
```

### Bash Completion

```bash
eval "$(nitric completion bash)"
```