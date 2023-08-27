#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
	deb-systemd-invoke stop wg-ui.service >/dev/null || true
fi
