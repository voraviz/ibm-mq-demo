# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1784190466 AS builder

# Allow the Go toolchain manager to download Go 1.26.4 as required by go.mod
ENV GOTOOLCHAIN=auto

# IBM MQ redistributable client. IBM publishes it for LinuxX64 only (no ARM64),
# so this image is amd64-only. Provides cmqc.h + libmqm_r.so for the mq-golang
# CGO bindings (mq-golang-jms20 wraps mq-golang/v5, which requires CGO).
ARG MQ_REDIST_URL=https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.4.5.1-IBM-MQC-Redist-LinuxX64.tar.gz
ARG MQ_INSTALL_DIR=/opt/mqm
USER root

RUN mkdir -p ${MQ_INSTALL_DIR} \
    && curl -fsSL "${MQ_REDIST_URL}" | tar -xz -C "${MQ_INSTALL_DIR}"

WORKDIR /build

# Cache Go module downloads separately from source compilation
COPY go.mod go.sum ./
RUN go mod download

# Compile. CGO is mandatory — mq-golang binds to the MQ C client. The binary is
# therefore dynamically linked against libmqm_r.so and glibc.
COPY . .
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="-I${MQ_INSTALL_DIR}/inc -D_REENTRANT" \
    CGO_LDFLAGS="-L${MQ_INSTALL_DIR}/lib64 -lmqm_r -Wl,-rpath,${MQ_INSTALL_DIR}/lib64" \
    go build -o /api-app-go-jms20 .

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM registry.redhat.io/ubi9/ubi-minimal:latest
ARG MQ_INSTALL_DIR=/opt/mqm

# MQ client shared libraries. Copy the whole dir — the client dlopen's its
# transport plugins at runtime, so they aren't captured by ldd alone.
COPY --from=builder ${MQ_INSTALL_DIR}/lib64 ${MQ_INSTALL_DIR}/lib64

# GSKit — IBM's TLS/SSL engine. Required for TLS-secured MQ channels; the MQ
# client dlopen's libgsk*/libick* from here using its install-relative layout,
# so it must sit at the same path. ldd on lib64 alone does NOT reveal these.
COPY --from=builder ${MQ_INSTALL_DIR}/gskit9 ${MQ_INSTALL_DIR}/gskit9

# ubi-minimal ships glibc + NSS modules, but not the C++ runtime that some MQ
# client libraries link against. Bring those two over from the builder.
COPY --from=builder /usr/lib64/libstdc++.so.6 /usr/lib64/libgcc_s.so.1 /usr/lib64/

COPY --from=builder /api-app-go-jms20 /app/api-app-go-jms20
USER root

RUN mkdir -p /IBM \
    && chown 1001 /IBM
EXPOSE 8081

USER 1001

ENV HOME=/IBM \
    SERVER_PORT=8081 \
    LD_LIBRARY_PATH=/opt/mqm/lib64:/opt/mqm/gskit9/lib64

ENTRYPOINT ["/app/api-app-go-jms20"]
