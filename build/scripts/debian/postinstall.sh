#!/bin/sh
set -e

if [ "$1" = "configure" ]; then
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
fi

if [ "$1" = "configure" ] || [ "$1" = "abort-upgrade" ] || [ "$1" = "abort-deconfigure" ] || [ "$1" = "abort-remove" ] ; then
	# This will only remove masks created by d-s-h on package removal.
	deb-systemd-helper unmask wg-ui.service >/dev/null || true

	# was-enabled defaults to true, so new installations run enable.
	if deb-systemd-helper --quiet was-enabled wg-ui.service; then
		# Enables the unit on first installation, creates new
		# symlinks on upgrades if the unit file has changed.
		deb-systemd-helper enable wg-ui.service >/dev/null || true
		deb-systemd-invoke start wg-ui.service >/dev/null || true
	else
		# Update the statefile to add new symlinks (if any), which need to be
		# cleaned up on purge. Also remove old symlinks.
		deb-systemd-helper update-state wg-ui.service >/dev/null || true
	fi

	# Restart only if it was already started
	if [ -d /run/systemd/system ]; then
		systemctl --system daemon-reload >/dev/null || true
		if [ -n "$2" ]; then
			deb-systemd-invoke try-restart wg-ui.service >/dev/null || true
		fi
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
