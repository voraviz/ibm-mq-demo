package mq

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/config"
	"github.com/ibm-messaging/mq-golang-jms20/jms20subset"
)

// Consumer reads messages from an IBM MQ queue in a background goroutine,
// broadcasting each received message via a configurable callback.
// Mirrors MQConsumer.java exactly: 500ms poll loop, 1s reconnect retry.
type Consumer struct {
	cf          jms20subset.ConnectionFactory
	cfg         *config.Config
	broadcaster func(string)

	mu       sync.Mutex
	jmsCtx   jms20subset.JMSContext
	jmsCons  jms20subset.JMSConsumer
	stopping bool
	started  bool

	counter atomic.Int64
}

// NewConsumer creates a Consumer. Call SetBroadcaster before Start.
func NewConsumer(cfg *config.Config) *Consumer {
	cf := newConnectionFactory(cfg)
	return &Consumer{cf: cf, cfg: cfg}
}

// SetBroadcaster registers the function called with each received message text.
func (c *Consumer) SetBroadcaster(fn func(string)) {
	c.broadcaster = fn
}

// Start opens the JMS connection and launches the read goroutine.
// Idempotent — safe to call when already running.
func (c *Consumer) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return
	}
	if err := c.openResources(); err != nil {
		log.Printf("MQ consumer: failed to start: %v", err)
		return
	}
	c.stopping = false
	c.started = true
	go c.readLoop()
	log.Printf("MQ consumer started — listening on queue: %s", c.cfg.Queue)
}

// Stop signals the read goroutine to exit and closes JMS resources.
// Idempotent — safe to call when already stopped.
func (c *Consumer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}
	c.stopping = true
	c.closeResources()
	c.started = false
	log.Println("MQ consumer stopped")
}

// IsRunning reports whether the consumer goroutine is active.
func (c *Consumer) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.started
}

// GetCount returns the total number of messages received from the queue.
func (c *Consumer) GetCount() int64 {
	return c.counter.Load()
}

// openResources creates the JMSContext and JMSConsumer.
// Must be called with c.mu held.
func (c *Consumer) openResources() error {
	ctx, jmsErr := c.cf.CreateContext(multiHostOption(c.cfg))
	if jmsErr != nil {
		return jmsErr
	}

	dest := ctx.CreateQueue(c.cfg.Queue)

	cons, jmsErr := ctx.CreateConsumer(dest)
	if jmsErr != nil {
		ctx.Close()
		return jmsErr
	}

	c.jmsCtx = ctx
	c.jmsCons = cons
	return nil
}

// closeResources closes the JMSConsumer and JMSContext.
// Must be called with c.mu held.
func (c *Consumer) closeResources() {
	if c.jmsCons != nil {
		c.jmsCons.Close()
		c.jmsCons = nil
	}
	if c.jmsCtx != nil {
		c.jmsCtx.Close()
		c.jmsCtx = nil
	}
}

// readLoop polls the queue every 500ms (matching the Java 500ms JMS receive
// timeout). On JMS error it calls reconnectLoop; on stop signal it exits.
func (c *Consumer) readLoop() {
	for {
		c.mu.Lock()
		stopping := c.stopping
		cons := c.jmsCons
		c.mu.Unlock()

		if stopping || cons == nil {
			return
		}

		// Receive with 500ms timeout — mirrors jmsConsumer.receive(500) in MQConsumer.java.
		// A nil message with nil error means no message was available (timeout expired).
		msg, jmsErr := cons.Receive(500)

		if jmsErr != nil {
			// Check if we're stopping before attempting reconnect.
			c.mu.Lock()
			stopping = c.stopping
			c.mu.Unlock()
			if stopping {
				return
			}
			log.Printf("MQ consumer disconnected: %v — reconnecting", jmsErr)
			c.reconnectLoop()
			continue
		}

		if msg == nil {
			// Timeout with no message — loop and check stop flag.
			continue
		}

		// Extract the text body. The JMS20 library wraps messages as TextMessage.
		var text string
		if tm, ok := msg.(jms20subset.TextMessage); ok {
			if t := tm.GetText(); t != nil {
				text = *t
			}
		}

		log.Printf("GET message: %s", text)
		if c.broadcaster != nil {
			c.broadcaster(text)
		}
		c.counter.Add(1)
	}
}

// reconnectLoop closes JMS resources and retries every 1 second until
// the connection is re-established or Stop() is called.
// Mirrors MQConsumer.java reconnectLoop().
func (c *Consumer) reconnectLoop() {
	for {
		c.mu.Lock()
		if c.stopping {
			c.mu.Unlock()
			return
		}
		c.closeResources()
		err := c.openResources()
		c.mu.Unlock()

		if err == nil {
			log.Println("MQ consumer reconnected")
			return
		}
		log.Printf("MQ consumer reconnect failed: %v", err)
		time.Sleep(1 * time.Second)
	}
}
