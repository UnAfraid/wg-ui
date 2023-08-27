#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
	systemctl --system daemon-reload >/dev/null || true
fi

systemctl mask wg-ui.service >/dev/null || true
systemctl purge wg-ui.service >/dev/null || true
systemctl unmask wg-ui.service >/dev/null || true
