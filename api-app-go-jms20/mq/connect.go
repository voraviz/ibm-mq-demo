package mq

import (
	"fmt"
	"log"

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
		QMName:      cfg.QueueManager,
		Hostname:    host,
		PortNumber:  port,
		ChannelName: cfg.Channel,
		UserName:    cfg.Username,
		Password:    cfg.Password,
	}

	if cfg.CcdtUrl != "" {
		log.Printf("MQ connect: qmgr=%s ccdt=%s user=%s",
			cfg.QueueManager, cfg.CcdtUrl, cfg.Username)
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

// connectOption returns an MQOptions callback that applies the correct
// connection strategy at connect time:
//   - CCDT: sets cno.CCDTUrl; channel and connection list come from the file.
//   - Connection list: overwrites cno.ClientConn.ConnectionName with the full
//     multi-host list and enables MQCNO_RECONNECT_Q_MGR for automatic HA failover.
func connectOption(cfg *config.Config) jms20subset.MQOptions {
	return func(cno *ibmmq.MQCNO) {
		cno.Options |= ibmmq.MQCNO_RECONNECT_Q_MGR
		if cfg.CcdtUrl != "" {
			// CCDT takes precedence — channel and connection list are read
			// from the CCDT file; no ConnectionName override is needed.
			cno.CCDTUrl = cfg.CcdtUrl
		} else {
			// Override ConnectionName with the full multi-host list.
			cno.ClientConn.ConnectionName = connectionName(cfg)
			cno.ClientConn.HeartbeatInterval = int32(cfg.ReconnectTimeout)
		}
	}
}
