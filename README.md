# Nitric CLI

![GitHub Workflow Status (branch)](https://img.shields.io/github/workflow/status/nitrictech/cli/Tests/develop?label=tests)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/nitrictech/cli)
![GitHub all releases](https://img.shields.io/github/downloads/nitrictech/cli/total)

## Overview

Nitric provides a command-line interface (CLI) to assist with various tasks when building applications with the Nitric framework. For more information, check out the main [Nitric repo](https://github.com/nitrictech/nitric).

## Installation

The Nitric CLI is free to [download and install](https://nitric.io/docs/installation).

## Purpose

The Nitric CLI performs 3 main tasks:

- Create new projects
- Run apps locally for testing and development
- Deploy to the cloud you choose

## Common Commands

Common commands in the CLI that you’ll be using:

- nitric down : Undeploy a previously deployed stack, deleting resources
- nitric run : Run your project locally for development and testing
- nitric stack new : Create a new Nitric stack
- nitric start : Run nitric services locally for development and testing
- nitric up : Create or update a deployed stack

## Help with Commands

Each command is self documented and provides a "help" interface describing the usage, arguments and options for the command. Use the help command to view the help information for any other command:

Example displaying help for the `new` command

```bash
nitric new --help
```

## Complete Reference

Documentation for all available commands:

- nitric feedback : Provide feedback on your experience with nitric
- nitric info : Gather information about Nitric and the environment
- nitric new [projectName] [templateName] : Create a new project
- nitric run : Run your project locally for development and testing
- nitric stack : Manage stacks (the deployed app containing multiple resources e.g. collection, bucket, topic)
- nitric stack down [-s stack] : Undeploy a previously deployed stack, deleting resources
  (alias: nitric down)
- nitric stack list [-s stack] : List all project stacks and their status
  (alias: nitric list)
- nitric stack new : Create a new Nitric stack
- nitric stack update [-s stack] : Create or update a deployed stack
  (alias: nitric up)
- nitric start : Run nitric services locally for development and testing
- nitric version : Print the version number of this CLI

## Get in touch

- Ask questions in [GitHub discussions](https://github.com/nitrictech/nitric/discussions)

- Find us on [Twitter](https://twitter.com/nitric_io)

- Send us an [email](mailto:maintainers@nitric.io)

## Get Started

Check out the [Nitric docs](https://nitric.io/docs) to see how to get started using Nitric.
cs](https://nitric.io/docs) to see how to get started using Nitric.
using Nitric.
