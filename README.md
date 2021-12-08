<p align="center">
  <img src="./logo.png" alt="Nitric Logo"/>
</p>

# Nitric CLI

## Overview

Nitric provides a command-line interface (CLI) to assist with various tasks when building applications with Nitric. To view a list of available commands, you can call the CLI without specifying any subcommands or arguments:

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