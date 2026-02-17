# wg-ui
[![Go](https://github.com/UnAfraid/wg-ui/actions/workflows/go.yml/badge.svg)](https://github.com/UnAfraid/wg-ui/actions/workflows/go.yml)

Self-contained WireGuard management service with a web UI and GraphQL API written in pure Go.

## Features
* Ability to import existing wireguard configurations
* Web UI - https://github.com/desislavsd/wireguard-manager
* Multiple wireguard interfaces
* Simple multi-user authentication support
* Portable - No external dependencies (one single binary)
* Multiple backend support:
  * **Linux**: Direct kernel module communication via netlink
  * **NetworkManager**: Integration with NetworkManager (Linux)
  * **macOS**: Userspace WireGuard via wireguard-go
* Flexible deployment - binary and oci container

## Platform Support

| Platform | Backend | WireGuard Mode | Status |
|----------|---------|----------------|--------|
| Linux | `linux` | Kernel module | ✅ Stable |
| Linux | `networkmanager` | NetworkManager | ✅ Stable |
| macOS | `darwin` | Userspace (wireguard-go) | ✅ New |
| Windows | - | - | ❌ Not supported |

### macOS Limitations
* Interface names must use `utun` prefix (automatically normalized)
* Firewall marks not supported (Darwin OS limitation)
* Requires root/admin privileges
* Userspace implementation (lower performance than kernel module)

# Quickstart
Compile from source or download a release from [Releases](https://github.com/UnAfraid/wg-ui/releases/latest) page

# Installation

## macOS
### Using Homebrew (recommended)
```shell
# Install WireGuard tools (optional, for wg command)
brew install wireguard-tools

# Download and install wg-ui
# (Replace with actual download URL once released)
curl -L https://github.com/UnAfraid/wg-ui/releases/latest/download/wg-ui_darwin_amd64 -o wg-ui
chmod +x wg-ui

# Copy default .env.dist as .env
curl -L https://raw.githubusercontent.com/UnAfraid/wg-ui/master/.env.dist -o .env

# Generate random jwt secret (macOS)
sed -i '' "s|WG_UI_JWT_SECRET=Any_secret_base64_value_here|WG_UI_JWT_SECRET=$(head -c 128 </dev/urandom | base64)|g" .env

# Start wg-ui (requires root for network operations)
sudo env $(cat .env | xargs) ./wg-ui
```

### Requirements
* macOS 11.0 (Big Sur) or later
* Root/administrator privileges
* Xcode Command Line Tools (for building from source)

**Note**: The macOS backend uses userspace WireGuard. Interface names will be automatically normalized to `utun` format (e.g., `wg0` becomes `utun0`).

## Linux

### Run manually from the shell
```shell
# Copy default .env.dist as .env
cp .env.dist .env

# Generate random jwt secret
sed -i "s|WG_UI_JWT_SECRET=Any_secret_base64_value_here|WG_UI_JWT_SECRET=$(head -c 128 </dev/urandom | base64 -w 0 -)|g" .env

# Start wg-ui
env $(cat .env | xargs) ./wg-ui
```

# Debian/Ubuntu
### amd64 (x64)
```shell
sudo dpkg -i wg-ui_*_linux_amd64.deb
```
### arm64
```shell
sudo dpkg -i wg-ui_*_linux_arm64.deb
```


# Archlinux
### amd64 (x64)
```shell
sudo pacman -U --noconfirm wg-ui_*_linux_amd64.pkg.tar.zst
```
### arm64
```shell
sudo pacman -U --noconfirm wg-ui_*_linux_arm64.pkg.tar.zst
```

# Docker
```shell
# Download the docker-compose file
wget https://raw.githubusercontent.com/UnAfraid/wg-ui/master/docker-compose.yaml

# Download the default .env file
wget -O .env https://raw.githubusercontent.com/UnAfraid/wg-ui/master/.env.dist

# Generate random jwt secret
sed -i "s|WG_UI_JWT_SECRET=Any_secret_base64_value_here|WG_UI_JWT_SECRET=$(head -c 128 </dev/urandom | base64 -w 0 -)|g" .env

# Modify .env
nano .env

# Start the container
docker-compose up -d
```
