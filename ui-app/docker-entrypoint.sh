#!/bin/sh
set -e
# Backend the SPA's /api and /ws are proxied to. Override at runtime:
#   podman run -e API_UPSTREAM=http://host.containers.internal:8081 ...
: "${API_UPSTREAM:=http://localhost:8081}"

# Substitute the one placeholder (not envsubst — that would eat nginx's own
# $http_upgrade/$host runtime vars). Write to /tmp: the only dir 1001 can write.
sed "s|__API_UPSTREAM__|${API_UPSTREAM}|g" /opt/app-root/etc/nginx.conf.template > /tmp/nginx.conf

exec nginx -c /tmp/nginx.conf -g "daemon off;"
