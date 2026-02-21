# k8s-yaml-splitter

[![CI](https://github.com/nathanmartins/k8s-yaml-splitter/actions/workflows/ci.yaml/badge.svg)](https://github.com/nathanmartins/k8s-yaml-splitter/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nathanmartins/k8s-yaml-splitter)](https://goreportcard.com/report/github.com/nathanmartins/k8s-yaml-splitter)
[![License](https://img.shields.io/github/license/nathanmartins/k8s-yaml-splitter)](https://github.com/nathanmartins/k8s-yaml-splitter/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/nathanmartins/k8s-yaml-splitter)](https://github.com/nathanmartins/k8s-yaml-splitter/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nathanmartins/k8s-yaml-splitter)](https://github.com/nathanmartins/k8s-yaml-splitter/blob/main/go.mod)

`k8s-yaml-splitter`, is a powerful command-line interface (CLI) tool built for extraction of Kubernetes YAML manifests into individual files.

## Examples

It can work just by piping in from STDIN:

`kustomize build example/ | k8s-yaml-splitter out/`

## Installation

In order to use the `k8s-yaml-splitter` command-line tool, you need to have Go (version 1.21) installed on your system.

`go install github.com/nathanmartins/k8s-yaml-splitter@latest`

or download the binary over at:

https://github.com/nathanmartins/k8s-yaml-splitter/releases/latest
