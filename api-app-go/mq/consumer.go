package mq

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// Consumer reads messages from an IBM MQ queue in a background goroutine,
// broadcasting each received message via a configurable callback.
// Mirrors MQConsumer.java exactly: 500ms poll loop, 1s reconnect retry.
type Consumer struct {
	cfg         *config.Config
	broadcaster func(string)

	mu       sync.Mutex
	qmgr     *ibmmq.MQQueueManager
	qObj     *ibmmq.MQObject
	stopping bool
	started  bool

	counter atomic.Int64
}

// NewConsumer creates a Consumer. Call SetBroadcaster before Start.
func NewConsumer(cfg *config.Config) *Consumer {
	return &Consumer{cfg: cfg}
}

// SetBroadcaster registers the function called with each received message text.
func (c *Consumer) SetBroadcaster(fn func(string)) {
	c.broadcaster = fn
}

// Start opens the MQ connection and launches the read goroutine.
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

// Stop signals the read goroutine to exit and closes MQ resources.
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

// openResources creates the MQ connection, session and consumer handle.
// Must be called with c.mu held.
func (c *Consumer) openResources() error {
	qmgr, err := connect(c.cfg)
	if err != nil {
		return err
	}

	od := ibmmq.NewMQOD()
	od.ObjectType = ibmmq.MQOT_Q
	od.ObjectName = c.cfg.Queue

	qObj, err := qmgr.Open(od, ibmmq.MQOO_INPUT_AS_Q_DEF)
	if err != nil {
		qmgr.Disc()
		return err
	}

	c.qmgr = &qmgr
	c.qObj = &qObj
	return nil
}

// closeResources closes the queue object and disconnects.
// Must be called with c.mu held.
func (c *Consumer) closeResources() {
	if c.qObj != nil {
		c.qObj.Close(0)
		c.qObj = nil
	}
	if c.qmgr != nil {
		c.qmgr.Disc()
		c.qmgr = nil
	}
}

// readLoop polls the queue every 500ms (matching the Java 500ms JMS receive
// timeout). On MQ error it calls reconnectLoop; on stop signal it exits.
func (c *Consumer) readLoop() {
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_WAIT | ibmmq.MQGMO_CONVERT
	gmo.WaitInterval = 500 // milliseconds — mirrors MQConsumer.java jmsConsumer.receive(500)

	buf := make([]byte, 32768)

	for {
		c.mu.Lock()
		stopping := c.stopping
		qObj := c.qObj
		c.mu.Unlock()

		if stopping || qObj == nil {
			return
		}

		md := ibmmq.NewMQMD()
		datalen, err := qObj.Get(md, gmo, buf)

		if err != nil {
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				// Timeout with no message — loop and check stop flag
				continue
			}
			// Unexpected error — check if we're stopping
			c.mu.Lock()
			stopping = c.stopping
			c.mu.Unlock()
			if stopping {
				return
			}
			log.Printf("MQ consumer disconnected: %v — reconnecting", err)
			c.reconnectLoop()
			// After reconnect, refresh gmo loop variables
			continue
		}

		text := string(buf[:datalen])
		log.Printf("GET message: %s", text)
		if c.broadcaster != nil {
			c.broadcaster(text)
		}
		c.counter.Add(1)
	}
}

// reconnectLoop closes MQ resources and retries every 1 second until
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
