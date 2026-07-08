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
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING

	cd := ibmmq.NewMQCD()
	cd.ChannelName = cfg.Channel
	if cfg.ConnectionList != "" {
		cd.ConnectionName = cfg.ConnectionList
	} else {
		cd.ConnectionName = fmt.Sprintf("%s(%d)", cfg.Host, cfg.Port)
	}
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
