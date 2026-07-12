package mq

import (
	"fmt"
	"sync/atomic"

	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/config"
	"github.com/ibm-messaging/mq-golang-jms20/jms20subset"
)

// Producer sends messages to an IBM MQ queue using the JMS20 API.
// A fresh JMSContext is opened per Put call (stateless), matching the
// SmallRye Reactive Messaging emitter pattern in the Java app.
type Producer struct {
	cf  jms20subset.ConnectionFactory
	cfg *config.Config

	counter atomic.Int64
}

// NewProducer creates a Producer backed by a JMS20 connection factory.
func NewProducer(cfg *config.Config) *Producer {
	cf := newConnectionFactory(cfg)
	return &Producer{cf: cf, cfg: cfg}
}

// Put sends text to the configured queue.
// The message body is formatted as "[#N] text" where N is a monotonically
// incrementing per-process counter, matching MessageResource.java.
// Returns the formatted message body on success.
func (p *Producer) Put(text string) (string, error) {
	n := p.counter.Add(1)
	body := fmt.Sprintf("[#%d] %s", n, text)

	ctx, jmsErr := p.cf.CreateContext(connectOption(p.cfg))
	if jmsErr != nil {
		p.counter.Add(-1)
		return "", fmt.Errorf("MQ connect: %s", jmsErr.Error())
	}
	defer ctx.Close()

	dest := ctx.CreateQueue(p.cfg.Queue)

	if jmsErr := ctx.CreateProducer().SendString(dest, body); jmsErr != nil {
		p.counter.Add(-1)
		return "", fmt.Errorf("MQ send: %s", jmsErr.Error())
	}

	return body, nil
}

// PutCount returns the total number of successfully put messages.
func (p *Producer) PutCount() int64 {
	return p.counter.Load()
}
