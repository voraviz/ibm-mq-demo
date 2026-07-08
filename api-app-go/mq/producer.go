package mq

import (
	"fmt"
	"sync/atomic"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// Producer sends messages to an IBM MQ queue.
// A fresh MQ connection is opened per Put call (stateless), matching the
// SmallRye Reactive Messaging emitter pattern in the Java app.
type Producer struct {
	cfg     *config.Config
	counter atomic.Int64
}

// NewProducer creates a Producer.
func NewProducer(cfg *config.Config) *Producer {
	return &Producer{cfg: cfg}
}

// Put sends text to the configured queue.
// The message body is formatted as "[#N] text" where N is a monotonically
// incrementing per-process counter, matching MessageResource.java.
// Returns the formatted message body on success.
func (p *Producer) Put(text string) (string, error) {
	n := p.counter.Add(1)
	body := fmt.Sprintf("[#%d] %s", n, text)

	qmgr, _, err := connect(p.cfg)
	if err != nil {
		p.counter.Add(-1) // roll back on connection failure
		return "", fmt.Errorf("MQ connect: %w", err)
	}
	defer qmgr.Disc()

	od := ibmmq.NewMQOD()
	od.ObjectType = ibmmq.MQOT_Q
	od.ObjectName = p.cfg.Queue

	qObj, err := qmgr.Open(od, ibmmq.MQOO_OUTPUT)
	if err != nil {
		p.counter.Add(-1)
		return "", fmt.Errorf("MQ open queue: %w", err)
	}
	defer qObj.Close(0)

	md := ibmmq.NewMQMD()
	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

	if err := qObj.Put(md, pmo, []byte(body)); err != nil {
		p.counter.Add(-1)
		return "", fmt.Errorf("MQ put: %w", err)
	}

	return body, nil
}

// PutCount returns the total number of successfully put messages.
func (p *Producer) PutCount() int64 {
	return p.counter.Load()
}
