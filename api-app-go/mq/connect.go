package mq

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// resolveCcdtUrl normalizes a possibly-relative "file:" CCDT URL into an
// absolute "file:///..." form. The Go/C client requires an absolute path —
// a relative reference silently fails to load and produces MQRC 2058.
func resolveCcdtUrl(url string) string {
	const prefix = "file:"
	if !strings.HasPrefix(url, prefix) {
		return url
	}
	path := strings.TrimPrefix(url, prefix)
	if strings.HasPrefix(path, "///") {
		return url
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return url
	}
	return prefix + "///" + strings.TrimPrefix(abs, "/")
}

// connect opens a client connection to IBM MQ using MQCNO + MQCD + MQCSP.
// The caller is responsible for calling qmgr.Disc() when done.
func connect(cfg *config.Config) (ibmmq.MQQueueManager, string, error) {
	cno := ibmmq.NewMQCNO()
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING |
		// MQCNO_RECONNECT: allows reconnect within a Native HA group (failover)
		// AND across queue managers in a Uniform Cluster (balancing).
		// MQCNO_RECONNECT_Q_MGR would block cross-QM balancing entirely.
		ibmmq.MQCNO_RECONNECT

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
		cd.HeartbeatInterval = int32(cfg.ReconnectTimeout)
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
