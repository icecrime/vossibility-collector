vossibility-collector
---------------------

[![Circle CI](https://circleci.com/gh/icecrime/vossibility-collector.svg?style=svg)](https://circleci.com/gh/icecrime/vossibility-collector)

Vossibility-collector is the core component of [vossibility](https://github.com/icecrime/vossibility-stack),
a project providing better visibility for your open source project. The project was initially
started for [Docker](https://docker.io) but is not tied to it in any way.

# Overview

Vossibility-collector receives live GitHub data from a [NSQ](http://nsq.io/) queue and the [GitHub
API](https://developer.github.com/v3/) on one end, and feeds structured data to Elastic Search on
the other end. It provides:

 - The power of Elastic Search to search into your repository (e.g., "give me all pull requests
   comments from a user who's name is approximately ice and contains LGTM").
 - The power of Kibana to build dashboards for your project: a basic example of what can be achieved
   is shown below.

![Sample dashboard](https://github.com/icecrime/vossibility-collector/raw/master/resources/screen_1.png)

# Usage

```
NAME:
   vossibility-collector - collect GitHub repository data

USAGE:
   vossibility-collector [global options] command [command options] [arguments...]
   
VERSION:
   0.1.0
   
COMMANDS:
   limits       get information about your GitHub API rate limits
   run          listen and process GitHub events
   sync         sync storage with the GitHub repositories
   sync_mapping sync the configuration definition with the store mappings
   sync_users   sync the user store with the information from a file
   help, h      Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   -c, --config "config.toml"   configuration file
   --debug                      enable debug output
   --debug-es                   enable debug output for elasticsearch queries
   --help, -h                   show help
   --version, -v                print the version
```

# Documentation

 Document   | Description
 -----------|--------------------------------------------------------------------------------------
 [`installation.md`](https://github.com/icecrime/vossibility-collector/blob/master/docs/installation.md)  | Installation guide
 [`configuration.md`](https://github.com/icecrime/vossibility-collector/blob/master/docs/configuration.md) | Configuring vossibility
