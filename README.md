# wg-ui
[![Go](https://github.com/UnAfraid/wg-ui/actions/workflows/go.yml/badge.svg)](https://github.com/UnAfraid/wg-ui/actions/workflows/go.yml)

Self-contained WireGuard management service with a web UI and GraphQL API written in pure Go.

## Features
* Ability to import existing wireguard configurations
* Web UI - https://github.com/desislavsd/wireguard-manager
* Multiple wireguard interfaces
* Simple multi-user authentication support
* Portable - No external dependencies (one single binary)
* Configures wireguard by directly communicating with kernel module
* Flexible deployment - binary and oci container
