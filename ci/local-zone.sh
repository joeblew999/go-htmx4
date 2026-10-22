#!/usr/bin/env bash
# The machine's IANA time zone ("Asia/Bangkok"), which the local server passes the app as Cloudflare would
# (request.cf.timezone): mise run serve and mise run dev start in the developer's zone. TZ if set, else what
# /etc/localtime points at (macOS, most Linux), else /etc/timezone, else UTC.
set -euo pipefail
if [ -n "${TZ:-}" ]; then
  echo "${TZ#:}"
elif z=$(readlink /etc/localtime 2>/dev/null) && [[ $z == *zoneinfo/* ]]; then
  echo "${z##*zoneinfo/}"
elif [ -s /etc/timezone ]; then
  cat /etc/timezone
else
  echo UTC
fi
