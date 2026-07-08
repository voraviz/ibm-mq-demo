package mq

import (
	"fmt"
	"log"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// connect opens a client connection to IBM MQ using MQCNO + MQCD + MQCSP.
// The caller is responsible for calling qmgr.Disc() when done.
func connect(cfg *config.Config) (ibmmq.MQQueueManager, string, error) {
	cno := ibmmq.NewMQCNO()
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING |
		// MQCNO_RECONNECT_Q_MGR: on disconnect, the client library automatically
		// retries every address in ConnectionName (IBM_MQ_CONNECTION_LIST) to
		// find the active native HA node — Get/Put calls block transparently
		// during failover and resume without application-level retry.
		// Use this (not MQCNO_RECONNECT) when ConnectionName is a multi-host list.
		ibmmq.MQCNO_RECONNECT_Q_MGR

	cd := ibmmq.NewMQCD()
	cd.ChannelName = cfg.Channel
	if cfg.ConnectionList != "" {
		cd.ConnectionName = cfg.ConnectionList
	} else {
		cd.ConnectionName = fmt.Sprintf("%s(%d)", cfg.Host, cfg.Port)
	}
	// HeartbeatInterval controls how quickly a dead or unresponsive connection
	// is detected. Setting it equal to the reconnect timeout (IBM_MQ_RECONNECT_TIMEOUT,
	// default 30 s) matches Java's WMQ_CLIENT_RECONNECT_TIMEOUT behaviour: a
	// stalled reconnect attempt is abandoned after this many seconds.
	cd.HeartbeatInterval = int32(cfg.ReconnectTimeout)
	cno.ClientConn = cd
	log.Printf("MQ connect: qmgr=%s conn=%s channel=%s user=%s", cfg.QueueManager, cd.ConnectionName, cd.ChannelName, cfg.Username)

	// Authentication
	csp := ibmmq.NewMQCSP()
	csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
	csp.UserId = cfg.Username
	csp.Password = cfg.Password
	cno.SecurityParms = csp

	qmgr, err := ibmmq.Connx(cfg.QueueManager, cno)
	return qmgr, cd.ConnectionName, err
}
