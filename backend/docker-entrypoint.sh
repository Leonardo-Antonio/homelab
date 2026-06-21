#!/bin/sh
set -eu

mkdir -p /app/data/photos
chown -R homelab:homelab /app/data

exec su-exec homelab "$@"
