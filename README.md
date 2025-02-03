<h1 align="center">Tenantfinder</h1>

<p align="center">
Tenantfinder is a tool to discover domains and subdomains that are connected to the same Active Directory / Microsoft Entra tenant.
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#examples">Examples</a> •
</p>

---

## Overview

A fast, non-intrusive tool that helps discover domains and subdomains that are in the same Active Directory / Microsoft Entra tenant ([ref](https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/getfederationinformation-operation-soap)). Perfect for security professionals, researchers, and developers who need to map out infrastructure without sending direct requests to targets. 

This project is heavily inspired by the awesome tools from [ProjectDiscovery](https://github.com/projectdiscovery) like [subfinder](https://github.com/projectdiscovery/subfinder), [urlfinder](https://github.com/projectdiscovery/urlfinder) and [tldfinder](https://github.com/projectdiscovery/tldfinder).

![example](https://github.com/user-attachments/assets/d13848fa-384d-4d8f-8513-7dd9a932ebc0)


## Features

- **Optimized for Speed** and resource efficiency
- **STDIN/OUT** support for easy integration into existing workflows

## Installation

tenantfinder requires **Go 1.21**. Install it using the following command or download a pre-compiled binary from the [releases page](https://github.com/upmux/tenantfinder/releases).

```sh
go install -v github.com/upmux/tenantfinder/cmd/tenantfinder@latest
```

## Usage

```sh
tenantfinder -h
```

This command displays help for tenantfinder. Below are some common switches and options.

```yaml
A tool for discovering related domains and subdomains.

Usage:
  tenantfinder [flags]

Flags:
INPUT:
   -d, -domain string[]  domains to find subdomains for

SOURCE:
   -s, -sources string[]           specific sources to use for discovery (-s aad). Use -ls to display all available sources.
   -es, -exclude-sources string[]  sources to exclude from enumeration (-es aad)
   -all                            use all sources for enumeration (slow)

RATE-LIMIT:
   -rl, -rate-limit int      maximum number of http requests to send per second (global)
   -rls, -rate-limits value  maximum number of http requests to send per second four providers in key=value format (-rls aad=10/m) (default ["aad=10/m"])

OUTPUT:
   -o, -output string       file to write output to
   -j, -jsonl               write output in JSONL(ines) format
   -od, -output-dir string  directory to write output file
   -cs, -collect-sources    include all sources in the output (-json only)

CONFIGURATION:
   -proxy string  http proxy to use with tenantfinder

DEBUG:
   -silent             show only domains in output
   -version            show version of tenantfinder
   -v                  show verbose output
   -nc, -no-color      disable color in output
   -ls, -list-sources  list all available sources
   -stats              report source statistics

OPTIMIZATION:
   -timeout int   seconds to wait before timing out (default 30)
   -max-time int  minutes to wait for enumeration results (default 10)

```

## Examples

### Basic Usage

```console
tenantfinder -d tesla.com
```

This command enumerates domains for the target domain tesla.com.

Example run:

```console
$ tenantfinder -d tesla.com


  ______                       __  _______           __         
 /_  __/__  ____  ____ _____  / /_/ ____(_)___  ____/ /__  _____
  / / / _ \/ __ \/ __ \/ __ \/ __/ /_  / / __ \/ __  / _ \/ ___/
 / / /  __/ / / / /_/ / / / / /_/ __/ / / / / / /_/ /  __/ /    
/_/  \___/_/ /_/\__,_/_/ /_/\__/_/   /_/_/ /_/\__,_/\___/_/     

                        upmux.com

[INF] Enumerating domains for tesla.com
service.tesla.com
teslaalerts.com
c.tesla.com
teslagrohmannautomation.de
solarcity.com
t.tesla.com
m.tesla.com
tesla.com
[INF] Found 8 subdomains for tesla.com in 1 second 757 milliseconds
```

### Workflow example

Use STDIN to pass in a list of domains to enumerate and use the `-j` or `--jsonl` flag to output results in JSONL (JSON Lines) format, where each line is a separate JSON object. This format is useful for processing large outputs in a structured way.

#### Command Example

```console
echo "tesla.com" | tenantfinder -j -silent
```

#### Example JSONL Output

```json
{"domain":"m.tesla.com","input":"tesla.com","source":"aad"}
{"domain":"tesla.com","input":"tesla.com","source":"aad"}
{"domain":"service.tesla.com","input":"tesla.com","source":"aad"}
{"domain":"teslaalerts.com","input":"tesla.com","source":"aad"}
{"domain":"c.tesla.com","input":"tesla.com","source":"aad"}
{"domain":"teslagrohmannautomation.de","input":"tesla.com","source":"aad"}
{"domain":"solarcity.com","input":"tesla.com","source":"aad"}
{"domain":"t.tesla.com","input":"tesla.com","source":"aad"}
```

Each JSON object contains:
- `domain`: The discovered domain.
- `input`: The target domain (e.g., `tesla.com`).
- `source`: The data source for the domain discovery (e.g., `aad`).

--------

<div align="center">
  tenantfinder is made with ❤️ by [Upmux Security](https://upmux.com) and distributed under [MIT License](LICENSE).
</div>
