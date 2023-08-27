#!/bin/sh
set -e

# Add user and group
if ! getent group wg-ui >/dev/null; then
  groupadd --system wg-ui
fi

if ! getent passwd wg-ui >/dev/null; then
  useradd --system \
    --gid wg-ui \
    --create-home \
    --home-dir /var/lib/wg-ui \
    --shell /usr/sbin/nologin \
    --comment "wireguard manager" \
    wg-ui
fi

sed -i "s|WG_UI_JWT_SECRET=Any_secret_base64_value_here|WG_UI_JWT_SECRET=$(head -c 128 </dev/urandom | base64 -w 0 -)|g" /etc/default/wg-ui

# This will only remove masks created by d-s-h on package removal.
systemctl unmask wg-ui.service >/dev/null || true

# Restart only if it was already started
if [ -d /run/systemd/system ]; then
  systemctl --system daemon-reload >/dev/null || true
  if [ -n "$2" ]; then
    systemctl try-restart wg-ui.service >/dev/null || true
  fi
fi

cat <<EOF
wg-ui has been installed as a systemd service.

To start/stop wg-ui:

sudo systemctl start wg-ui
sudo systemctl stop wg-ui

To enable/disable wg-ui starting automatically on boot:

sudo systemctl enable wg-ui
sudo systemctl disable wg-ui

To view wg-ui logs:
journalctl -f -u wg-ui

To edit configuration modify /etc/default/wg-ui
When service is started visit http://localhost:4580

You can find the credentials to login in logs of wg-ui by running:
journalctl -u wg-ui | grep "admin user created" | awk -F': ' '{print \$2}'

EOF
