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

# Quickstart
Compile from source or download a release from [Releases](https://github.com/UnAfraid/wg-ui/releases/latest) page

# Run manually from the shell
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
