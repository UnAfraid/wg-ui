#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
	systemctl --system daemon-reload >/dev/null || true
fi

if [ "$1" = "remove" ]; then
	if [ -x "/usr/bin/deb-systemd-helper" ]; then
		deb-systemd-helper mask wg-ui.service >/dev/null || true
	else
	  systemctl mask wg-ui.service >/dev/null || true
	fi
fi

if [ "$1" = "purge" ]; then
	if [ -x "/usr/bin/deb-systemd-helper" ]; then
		deb-systemd-helper purge wg-ui.service >/dev/null || true
		deb-systemd-helper unmask wg-ui.service >/dev/null || true
	else
	  systemctl purge wg-ui.service >/dev/null || true
    systemctl unmask wg-ui.service >/dev/null || true
	fi
fi
