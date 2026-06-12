Examples using go-tfe 2.0
==============================

A collection of runnable examples that illustrate the use of the SDK client.

### Basic Usage

Build the example client:
```
$ go build -o tfe cmd/tfe/main.go
```

All subcommands can be explored using `--help` or by browsing each package directory
```
$ ./tfe account --help
```

All examples require setting the `TFE_TOKEN` and `TFE_ADDRESS` variables:
```
$ TFE_TOKEN=example TFE_ADDRESS=https://app.eu.terraform.io ./tfe account details
```
