# tasmota-proxy
tasmota-proxy

A simple CLI to control tasmota devices.

## Features

- something
- something else

## Installation

You can install a few ways:

1. Download the binary for your OS from https://github.com/smford/tasmota-proxy/releases
1. or use `go install`
   ```
   go install -v github.com/smford/tasmota-proxy@latest
   ```
1. or clone the git repo and build
   ```
   git clone git@github.com:smford/tasmota-proxy.git
   cd tasmota-proxy
   go get -v
   go build
   ```
1. or by brew:
   ```
   brew install smford/tap/tasmota-proxy
   ```

## Usage

1. By command line:
   `tasmota-proxy --device lamp --cmd status`
1. By configuration file:
   ```bash
   cat ~/.tasproxy
   ---
   verbose: true
   devices:
     poop: 111.11.11.1
     lamp: 172.28.10.12
     large: 192.168.10.127
   ```
1. By environment variable:
   `export TASPROXY_CONFIG="/path/to/config.yaml"`

## Command Line Options

```
--cmd [x]             Commands: on, off, status, statusall, timers
--config [file]       Configuration file: /path/to/file.yaml, default = ""/.tasproxy"
--custom [command]    Custom escaped command string to send
--device [name]       Name of device
--displayconfig       Display configuration
--help                Display help
--host [address]      IP address or hostname of device
--json                Output JSON
--list                List all configured devices
--verbose             Be verbose
--version             Display version
```

## Todo

- https compatability
- compatability with tasmota devices that need username and password
- sorting of device list
