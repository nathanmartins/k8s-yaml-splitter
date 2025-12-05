# k8s-yaml-splitter

`k8s-yaml-splitter`, is a powerful command-line interface (CLI) tool built for extraction of Kubernetes YAML manifests into individual files.

## Examples

It can work just by piping in from STDIN:

`kustomize build example/ | k8s-yaml-splitter out/`

## Installation

In order to use the `k8s-yaml-splitter` command-line tool, you need to have Go (version 1.21) installed on your system.

`go install github.com/nathanmartins/k8s-yaml-splitter@latest`

or download the binary over at:

https://github.com/nathanmartins/k8s-yaml-splitter/releases/latest
