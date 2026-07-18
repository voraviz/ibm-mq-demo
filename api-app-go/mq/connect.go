package mq

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// resolveCcdtUrl normalizes a CCDT location into an absolute "file:///..."
// URL. The Go/C MQ client requires an absolute path for file-based CCDTs —
// unlike the Java client, which resolves relative "file:" URLs against the
// JVM's working directory, a relative reference here silently fails to load,
// leaving no queue manager group defined and producing
// MQRC_Q_MGR_NAME_ERROR (2058) instead of a clear "file not found" error.
func resolveCcdtUrl(raw string) string {
	if raw == "" {
		return raw
	}

	// Non-file schemes (HTTP/HTTPS CCDT distribution) pass through untouched.
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}

	// Strip an optional "file:" prefix — a bare path with no scheme at all is
	// also valid input; the client assumes file: in that case.
	path := strings.TrimPrefix(raw, "file:")

	// Already a fully-qualified form — file:///path or file://host/path —
	// leave it alone.
	if strings.HasPrefix(path, "//") {
		return raw
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return raw // fall back; Connx will surface a clear error instead
	}
	abs = filepath.ToSlash(abs) // normalize \ to / for Windows paths

	if strings.HasPrefix(abs, "/") {
		return "file://" + abs // Unix absolute path already starts with "/"
	}
	return "file:///" + abs // Windows, e.g. "C:/Users/..." → file:///C:/Users/...
}

// connect opens a client connection to IBM MQ using MQCNO + MQCD + MQCSP.
// MQCNO_RECONNECT is set so the connection participates in Native HA failover
// and Uniform Cluster balancing. The caller must call qmgr.Disc() when done.
func connect(cfg *config.Config) (ibmmq.MQQueueManager, string, error) {
	return dial(cfg, true)
}

// connectNoReconnect opens a direct, non-reconnecting connection to a single
// host. Used only inside resolveActiveNode() probe loops: without
// MQCNO_RECONNECT a standby node immediately returns MQRC_STANDBY_Q_MGR
// (2539) instead of silently redirecting to the active node and masking the
// true host identity. The caller must call qmgr.Disc() when done.
func connectNoReconnect(cfg *config.Config) (ibmmq.MQQueueManager, string, error) {
	return dial(cfg, false)
}

// dial is the shared implementation for connect and connectNoReconnect.
func dial(cfg *config.Config, reconnect bool) (ibmmq.MQQueueManager, string, error) {
	cno := ibmmq.NewMQCNO()
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING
	if reconnect {
		// MQCNO_RECONNECT: allows reconnect within a Native HA group (failover)
		// AND across queue managers in a Uniform Cluster (balancing).
		// MQCNO_RECONNECT_Q_MGR would block cross-QM balancing entirely.
		cno.Options |= ibmmq.MQCNO_RECONNECT
	}

	connDesc := ""
	if cfg.CcdtUrl != "" {
		resolved := resolveCcdtUrl(cfg.CcdtUrl)
		cno.CCDTUrl = resolved
		connDesc = resolved
		log.Printf("MQ connect: qmgr=%s ccdt=%s user=%s", cfg.QueueManager, resolved, cfg.Username)
	} else {
		cd := ibmmq.NewMQCD()
		cd.ChannelName = cfg.Channel
		if cfg.ConnectionList != "" {
			cd.ConnectionName = cfg.ConnectionList
		} else {
			cd.ConnectionName = fmt.Sprintf("%s(%d)", cfg.Host, cfg.Port)
		}
		cd.HeartbeatInterval = int32(cfg.HeartbeatInterval)
		cno.ClientConn = cd
		connDesc = cd.ConnectionName
		log.Printf("MQ connect: qmgr=%s conn=%s channel=%s user=%s", cfg.QueueManager, cd.ConnectionName, cd.ChannelName, cfg.Username)
	}

	// ApplName identifies this app to MQ for Uniform Cluster balancing grouping
	// and DISPLAY APSTATUS('<name>') lookups.
	cno.ApplName = cfg.AppName

	// Authentication
	csp := ibmmq.NewMQCSP()
	csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
	csp.UserId = cfg.Username
	csp.Password = cfg.Password
	cno.SecurityParms = csp

	qmgr, err := ibmmq.Connx(cfg.QueueManager, cno)
	return qmgr, connDesc, err
}
