# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM registry.access.redhat.com/hi/go:latest AS builder

# Install the IBM MQ Redistributable Client (Linux x64).
# This provides cmqc.h and libmqm_r.so required by the mq-golang CGO bindings
# (mq-golang-jms20 wraps mq-golang/v5, which requires CGO).
# No root or package manager required — it is a plain tar archive.
ARG MQ_REDIST_URL=https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.4.5.1-IBM-MQC-Redist-LinuxX64.tar.gz
ARG MQ_INSTALL_DIR=/opt/mqm

RUN mkdir -p ${MQ_INSTALL_DIR} \
    && curl -fsSL "${MQ_REDIST_URL}" | tar -xz -C "${MQ_INSTALL_DIR}"

WORKDIR /build

# Cache Go module downloads separately from source compilation
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile.
# CGO_CFLAGS / CGO_LDFLAGS are set to the redistributable client location,
# matching the CGO directives in mq-golang/ibmmq/mqi.go.
COPY . .
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="-I${MQ_INSTALL_DIR}/inc -D_REENTRANT" \
    CGO_LDFLAGS="-L${MQ_INSTALL_DIR}/lib64 -lmqm_r -Wl,-rpath,${MQ_INSTALL_DIR}/lib64" \
    go build -o /api-app-go-jms20 .

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM registry.access.redhat.com/hi/go:latest

ARG MQ_INSTALL_DIR=/opt/mqm

# Copy only the shared libraries needed at runtime (libmqm_r and dependencies)
COPY --from=builder ${MQ_INSTALL_DIR}/lib64 ${MQ_INSTALL_DIR}/lib64

COPY --from=builder /api-app-go-jms20 /app/api-app-go-jms20

EXPOSE 8081

USER 65532

ENV SERVER_PORT=8081 \
    LD_LIBRARY_PATH=/opt/mqm/lib64

CMD ["/app/api-app-go-jms20"]
