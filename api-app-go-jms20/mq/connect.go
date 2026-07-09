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
// Hostname+PortNumber in its struct. For multi-host Native HA, we use an
// MQOptions callback to override cd.ConnectionName with the full connection
// list and set MQCNO_RECONNECT_Q_MGR so the client library automatically
// fails over to the active HA node — identical to api-app-go's connect.go.
func newConnectionFactory(cfg *config.Config) jms20subset.ConnectionFactory {
	// The primary host/port are used by ConnectionFactoryImpl to build the
	// initial MQCD. When a ConnectionList is provided, the MQOptions callback
	// in multiHostOption() overwrites ConnectionName with the full list.
	host := cfg.Host
	port := cfg.Port
	if cfg.ConnectionList != "" {
		// Parse the first entry of the list to satisfy ConnectionFactoryImpl's
		// required Hostname/PortNumber fields; the MQOptions callback replaces
		// the actual connection string at connect time.
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

	log.Printf("MQ connect: qmgr=%s conn=%s channel=%s user=%s",
		cfg.QueueManager, connectionName(cfg), cfg.Channel, cfg.Username)

	return cf
}

// connectionName returns the resolved ConnectionName string for logging.
func connectionName(cfg *config.Config) string {
	if cfg.ConnectionList != "" {
		return cfg.ConnectionList
	}
	return fmt.Sprintf("%s(%d)", cfg.Host, cfg.Port)
}

// multiHostOption returns an MQOptions function that overrides ConnectionName
// with the full multi-host list and enables MQCNO_RECONNECT_Q_MGR so the
// MQ client library automatically reconnects to the active HA node on failure.
func multiHostOption(cfg *config.Config) jms20subset.MQOptions {
	connName := connectionName(cfg)
	return func(cno *ibmmq.MQCNO) {
		// Override the ConnectionName set by ConnectionFactoryImpl with the
		// full multi-host list (e.g. "host1(1414),host2(1414),host3(1414)").
		cno.ClientConn.ConnectionName = connName
		// Enable automatic reconnect across the multi-host list.
		// MQCNO_RECONNECT_Q_MGR keeps the client on the same queue manager
		// across HA failover, matching api-app-go behaviour.
		cno.Options |= ibmmq.MQCNO_RECONNECT_Q_MGR
		// Match the heartbeat/reconnect timeout from config.
		cno.ClientConn.HeartbeatInterval = int32(cfg.ReconnectTimeout)
	}
}
