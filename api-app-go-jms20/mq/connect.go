package mq

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/config"
	"github.com/ibm-messaging/mq-golang-jms20/jms20subset"
	"github.com/ibm-messaging/mq-golang-jms20/mqjms"
	ibmmq "github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// newConnectionFactory builds a JMS20 ConnectionFactory from cfg.
//
// The JMS20 library's ConnectionFactoryImpl only supports a single
// Hostname+PortNumber in its struct. For multi-host Native HA or CCDT, we use
// an MQOptions callback (connectOption) to set the full connection details on
// the MQCNO at connect time — identical in behaviour to api-app-go's connect.go.
func newConnectionFactory(cfg *config.Config) jms20subset.ConnectionFactory {
	// ConnectionFactoryImpl requires a Hostname+PortNumber even when CCDT is
	// used — use the first entry of the connection list or the host/port config
	// as a placeholder; the connectOption callback overrides it at connect time.
	host := cfg.Host
	port := cfg.Port
	if cfg.ConnectionList != "" {
		if entries := parseConnEntries(cfg.ConnectionList); len(entries) > 0 {
			host = entries[0].host
			port = entries[0].port
		}
	}

	cf := mqjms.ConnectionFactoryImpl{
		// QMName should be the literal queue manager name for single-QM/Native HA
		// failover only (e.g. "QM1"). For Uniform Cluster balancing across queue
		// managers, this must be the queue manager GROUP name prefixed with "*"
		// (e.g. "*UNIQA") — see config.go / README for IBM_MQ_QUEUE_MANAGER.
		QMName:      cfg.QueueManager,
		Hostname:    host,
		PortNumber:  port,
		ChannelName: cfg.Channel,
		UserName:    cfg.Username,
		Password:    cfg.Password,
	}

	if cfg.CcdtUrl != "" {
		log.Printf("MQ connect: qmgr=%s ccdt=%s user=%s",
			cfg.QueueManager, resolveCcdtUrl(cfg.CcdtUrl), cfg.Username)
	} else {
		log.Printf("MQ connect: qmgr=%s conn=%s channel=%s user=%s",
			cfg.QueueManager, connectionName(cfg), cfg.Channel, cfg.Username)
	}

	return cf
}

// connectionName returns the resolved ConnectionName string for logging.
func connectionName(cfg *config.Config) string {
	if cfg.ConnectionList != "" {
		return cfg.ConnectionList
	}
	return fmt.Sprintf("%s(%d)", cfg.Host, cfg.Port)
}

// resolveCcdtUrl normalizes a possibly-relative "file:" CCDT URL into an
// absolute "file:///..." form. The Go/C client (unlike the Java client, which
// resolves relative file: URLs against the JVM working directory) requires an
// absolute path — a relative reference silently fails to load, leaving no
// queue manager group defined and producing MQRC_Q_MGR_NAME_ERROR (2058).
func resolveCcdtUrl(url string) string {
	const prefix = "file:"
	if !strings.HasPrefix(url, prefix) {
		return url // http(s):// or already some other scheme — leave as-is
	}
	path := strings.TrimPrefix(url, prefix)
	if strings.HasPrefix(path, "///") {
		return url // already absolute file:/// form
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return url // fall back to original; Connx will surface a clear error
	}
	return prefix + "///" + strings.TrimPrefix(abs, "/")
}

// connectOption returns an MQOptions callback that applies the correct
// connection strategy at connect time:
//   - CCDT: sets cno.CCDTUrl (resolved to an absolute path); channel and
//     connection list come from the file.
//   - Connection list: overwrites cno.ClientConn.ConnectionName with the full
//     multi-host list.
//   - MQCNO_RECONNECT (not _Q_MGR): allows both Native HA failover *and*
//     Uniform Cluster balancing to a different queue manager. _Q_MGR would
//     permit failover but block balancing entirely — DISPLAY APSTATUS would
//     show BALANCED(NOTAPPLIC)/MOVABLE(NO) with the connection stuck on QM1.
//   - ApplName: identifies this app to MQ for balancing grouping and
//     DISPLAY APSTATUS('<name>') lookups; previously unset.
func connectOption(cfg *config.Config) jms20subset.MQOptions {
	return func(cno *ibmmq.MQCNO) {
		cno.Options |= ibmmq.MQCNO_RECONNECT
		cno.ApplName = cfg.AppName

		if cfg.CcdtUrl != "" {
			// CCDT takes precedence — channel and connection list are read
			// from the CCDT file; no ConnectionName override is needed.
			cno.CCDTUrl = resolveCcdtUrl(cfg.CcdtUrl)
		} else {
			// Override ConnectionName with the full multi-host list.
			cno.ClientConn.ConnectionName = connectionName(cfg)
			cno.ClientConn.HeartbeatInterval = int32(cfg.ReconnectTimeout)
		}
	}
}
