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
// 500ms poll loop, 1s reconnect retry — same model as api-app-go. (The Java
// all-in-one-app uses an async JMS MessageListener instead; mq-golang-jms20
// has no async listener, so this app polls.)
type Consumer struct {
	cf          jms20subset.ConnectionFactory
	cfg         *config.Config
	broadcaster func(string)

	mu       sync.Mutex
	jmsCtx   jms20subset.JMSContext
	jmsCons  jms20subset.JMSConsumer
	stopping bool
	started  bool
	wg       sync.WaitGroup // tracks the read goroutine so Stop can wait for it

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
	c.wg.Add(1)
	go c.readLoop()
	log.Printf("MQ consumer started — listening on queue: %s", c.cfg.Queue)
}

// Stop signals the read goroutine to exit and waits for it to close the JMS
// resources it owns. The goroutine — never Stop — performs the JMSConsumer/
// JMSContext close, so JMS verbs are never issued against the context from two
// goroutines at once. Idempotent — safe to call when already stopped.
func (c *Consumer) Stop() {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return
	}
	c.stopping = true
	c.mu.Unlock()

	// Wait for readLoop to observe the flag, exit, and release the handles.
	// Bounded by the 500ms Receive wait (or ~1s reconnect backoff).
	c.wg.Wait()

	c.mu.Lock()
	c.started = false
	c.mu.Unlock()
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
	// Transacted → SESSIONTRANSACTED, so gets run under MQGMO_SYNCPOINT and the
	// message stays on the queue until readLoop commits (at-least-once). Default
	// (AUTOACKNOWLEDGE) is NO_SYNCPOINT — the message is removed on get. See
	// native-ha.md "Consumer delivery mode".
	var ctx jms20subset.JMSContext
	var jmsErr jms20subset.JMSException
	if c.cfg.Transacted {
		ctx, jmsErr = c.cf.CreateContextWithSessionMode(jms20subset.JMSContextSESSIONTRANSACTED, connectOption(c.cfg))
	} else {
		ctx, jmsErr = c.cf.CreateContext(connectOption(c.cfg))
	}
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
	defer c.wg.Done()
	// The read goroutine owns the JMS resources: it is the only place they are
	// closed, so Stop()/reconnect never race an in-flight Receive on them.
	defer func() {
		c.mu.Lock()
		c.closeResources()
		c.mu.Unlock()
	}()

	for {
		c.mu.Lock()
		stopping := c.stopping
		cons := c.jmsCons
		ctx := c.jmsCtx
		c.mu.Unlock()

		if stopping || cons == nil {
			return
		}

		// Receive with 500ms timeout — same poll cadence as api-app-go's qObj.Get.
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

		// Transacted mode: the message was retrieved under syncpoint. Commit
		// AFTER handing it off (broadcast) — a crash before here rolls back and
		// the message is redelivered (at-least-once). Committing per message
		// also closes the unit of work, so the connection returns to a MOVABLE
		// state between messages instead of staying INTRANS. On an idle queue no
		// message is retrieved, so no unit of work is opened and nothing to commit.
		if c.cfg.Transacted && ctx != nil {
			if commitErr := ctx.Commit(); commitErr != nil {
				log.Printf("MQ consumer commit failed: %v — message will be redelivered", commitErr)
			}
		}
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
