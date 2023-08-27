#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
	systemctl stop wg-ui.service >/dev/null || true
fi
