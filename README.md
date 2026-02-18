# wg-ui
[![Go](https://github.com/UnAfraid/wg-ui/actions/workflows/go.yml/badge.svg)](https://github.com/UnAfraid/wg-ui/actions/workflows/go.yml)

`wg-ui` is a WireGuard management service with:
- embedded web UI
- GraphQL API
- single Go binary deployment

## Highlights
- Import and manage existing WireGuard interfaces
- Manage multiple servers/interfaces and peers
- Multi-user auth with JWT sessions
- Automatic interface stats updates (`rxBytes`/`txBytes`)
- Works with different WireGuard backends (kernel, NetworkManager, exec)
- Deploy as a binary, package, or OCI container

## Backend Drivers
Backends are configured by URL scheme.

| Driver | OS | URL example                                                             | Privileges / requirements |
| --- | --- |-------------------------------------------------------------------------| --- |
| `linux` | Linux | `linux:///etc/wireguard`                                                | Needs `CAP_NET_ADMIN` (typically run as root) to create/configure interfaces directly via kernel APIs |
| `networkmanager` | Linux | `networkmanager:///`                                                    | Needs access to NetworkManager system DBus operations (typically root or polkit-authorized service user) |
| `exec` | Linux, macOS | `exec:///etc/wireguard?sudo=true`                                       | Uses `wg` / `wg-quick` and files under the configured path; with `sudo=true` it requires passwordless sudo for backend commands |
| `routeros` | Any OS (network reachability required) | `routeros://admin:secret@192.168.88.1:443/rest?insecureSkipVerify=true` | Uses RouterOS REST API over HTTPS to manage WireGuard interfaces and peers |

`exec` is useful when you want behavior close to native WireGuard CLI tooling.

### Exec backend sudo mode
If the app does not run as root, set:
- `exec:///etc/wireguard?sudo=true`

`sudo=true` is non-interactive, so passwordless sudo (`NOPASSWD`) is required for invoked commands.
- Linux commands: `wg`, `wg-quick`, `ip`, plus file operations needed for config management.
- macOS commands: `wg`, `wg-quick`, `ifconfig`, `route`, `netstat`, plus file operations needed for config management.

### RouterOS backend
`routeros` URL format:
- `routeros://user:password@host:port/rest`

Supported query parameters:
- `insecureSkipVerify=true` to ignore self-signed/invalid TLS certificates.

RouterOS backend always uses HTTPS REST API endpoints.

## Quickstart (Binary)
Download a release from [Releases](https://github.com/UnAfraid/wg-ui/releases/latest) or build locally:

```shell
go build -o wg-ui .
```

Initialize config and run:

```shell
# 1) Copy default config
cp .env.dist .env

# 2) Generate JWT secret and set it in .env (GNU/BSD sed compatible)
JWT_SECRET="$(openssl rand -base64 64 | tr -d '\n')"
sed -i.bak "s|^WG_UI_JWT_SECRET=.*|WG_UI_JWT_SECRET=${JWT_SECRET}|" .env && rm -f .env.bak

# 3) Start wg-ui with env vars from .env
set -a
. ./.env
set +a
./wg-ui
```

Default endpoints:
- App/UI: `http://127.0.0.1:4580`
- GraphQL: `http://127.0.0.1:4580/query`
- Health: `http://127.0.0.1:4580/health`

## Docker
```shell
# Download compose + env files
wget https://raw.githubusercontent.com/UnAfraid/wg-ui/master/docker-compose.yaml
wget -O .env https://raw.githubusercontent.com/UnAfraid/wg-ui/master/.env.dist

# Set JWT secret
JWT_SECRET="$(openssl rand -base64 64 | tr -d '\n')"
sed -i.bak "s|^WG_UI_JWT_SECRET=.*|WG_UI_JWT_SECRET=${JWT_SECRET}|" .env && rm -f .env.bak

# Start
docker compose up -d
```

## Linux Packages
Debian/Ubuntu:
```shell
sudo dpkg -i wg-ui_*_linux_amd64.deb
# or
sudo dpkg -i wg-ui_*_linux_arm64.deb
```

Arch Linux:
```shell
sudo pacman -U --noconfirm wg-ui_*_linux_amd64.pkg.tar.zst
# or
sudo pacman -U --noconfirm wg-ui_*_linux_arm64.pkg.tar.zst
```

## Configuration
Use `.env.dist` as the base. Important values:
- `WG_UI_JWT_SECRET`
- `WG_UI_HTTP_SERVER_HOST`
- `WG_UI_HTTP_SERVER_PORT`
- `WG_UI_BOLT_DB_PATH`
- `WG_UI_INITIAL_EMAIL`
- `WG_UI_INITIAL_PASSWORD`

See `.env.dist` for full configuration and defaults.
