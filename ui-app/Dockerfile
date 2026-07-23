# Static Vue SPA served by nginx, with /api and /ws reverse-proxied to the
# backend. The build (npm run build -> dist/) happens on the host in
# build_container.sh — doing it here would reintroduce the arm64 esbuild crash
# and break the build. So this is a pure copy of the prebuilt dist/.
#
# Build the SPA with an EMPTY VITE_API_BASE_URL so it calls same-origin /api and
# /ws; nginx proxies those to API_UPSTREAM (set at runtime, see entrypoint).
FROM registry.access.redhat.com/ubi9/nginx-124:latest

COPY dist/ /opt/app-root/src/
COPY nginx.conf.template /opt/app-root/etc/nginx.conf.template
COPY docker-entrypoint.sh /opt/app-root/etc/docker-entrypoint.sh

EXPOSE 8080

# The S2I base's default CMD only prints usage — run our entrypoint, which
# substitutes API_UPSTREAM into the config and starts nginx.
CMD ["/bin/sh", "/opt/app-root/etc/docker-entrypoint.sh"]
