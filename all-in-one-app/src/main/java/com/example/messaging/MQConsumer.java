package com.example.messaging;

import com.example.config.MQConfig;
import com.example.ws.MessageWebSocket;
import jakarta.annotation.PreDestroy;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import jakarta.jms.Connection;
import jakarta.jms.ConnectionFactory;
import jakarta.jms.JMSException;
import jakarta.jms.Message;
import jakarta.jms.MessageConsumer;
import jakarta.jms.Queue;
import jakarta.jms.Session;
import jakarta.jms.TextMessage;
import org.jboss.logging.Logger;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Async JMS consumer: the MQ JMS provider delivers messages to {@link #onMessage}
 * on its own thread. Between deliveries the connection sits idle in a MOVABLE
 * state, so a uniform cluster can rebalance it without any polling workaround —
 * unlike a manual blocking-receive loop, this needs no pulse interval.
 *
 * Two-layer reconnect: Layer 1 is the client library's WMQ_CLIENT_RECONNECT
 * (set on the ConnectionFactory), which absorbs most HA/cluster failovers
 * transparently. Layer 2 is {@link #onException} — the fallback that rebuilds
 * JMS resources when Layer 1 surfaces a connection error.
 */
@ApplicationScoped
public class MQConsumer {

    private static final Logger LOG = Logger.getLogger(MQConsumer.class);

    @Inject
    ConnectionFactory connectionFactory;

    @Inject
    MQConfig config;

    @Inject
    MessageWebSocket webSocket;

    // JMS resources — null when stopped/reconnecting
    private volatile Connection      jmsConnection;
    private volatile Session         jmsSession;
    private volatile MessageConsumer jmsConsumer;

    private volatile boolean running = false; // intent: started and not stopped
    private volatile boolean closing = false; // teardown in progress — suppress Layer-2 reconnect
    private final AtomicBoolean reconnecting = new AtomicBoolean(false);

    private final AtomicLong receiveCounter = new AtomicLong(0);

    public synchronized void start() {
        if (running) {
            return; // already running
        }
        try {
            openJmsResources();
            running = true;
            LOG.info("MQ consumer started — listening on queue: " + config.queue());
        } catch (JMSException e) {
            LOG.error("Failed to start MQ consumer", e);
            closeJmsResources();
        }
    }

    public synchronized void stop() {
        if (!running) {
            return; // already stopped
        }
        running = false;
        closeJmsResources();
        LOG.info("MQ consumer stopped");
    }

    public boolean isRunning() {
        return running;
    }

    public long getCount() {
        return receiveCounter.get();
    }

    @PreDestroy
    public synchronized void onDestroy() {
        if (running) {
            running = false;
            closeJmsResources();
            LOG.info("MQ consumer shut down on application stop");
        }
    }

    // ── private ──────────────────────────────────────────────────────────────

    /** Async delivery callback — acknowledged automatically when it returns. */
    private void onMessage(Message msg) {
        try {
            if (msg instanceof TextMessage textMsg) {
                String text = textMsg.getText();
                LOG.infof("GET message: %s", text);
                webSocket.broadcast(text);
                receiveCounter.incrementAndGet();
            } else {
                LOG.warnf("Ignoring non-text message of type: %s", msg.getClass().getName());
            }
        } catch (JMSException e) {
            LOG.error("Error handling MQ message", e);
        }
    }

    /** Layer-2 fallback: fired by the provider when a connection error surfaces. */
    private void onException(JMSException e) {
        if (!running || closing) {
            return; // stopping, or a teardown we triggered — ignore
        }
        if (!reconnecting.compareAndSet(false, true)) {
            return; // a reconnect is already in flight
        }
        LOG.warnf("MQ consumer connection error: %s — reconnecting", e.getLocalizedMessage());
        Thread.ofPlatform().name("mq-consumer-reconnect").daemon(true).start(() -> {
            try {
                reconnectLoop();
            } finally {
                reconnecting.set(false);
            }
        });
    }

    private void reconnectLoop() {
        while (running && !closing && !Thread.currentThread().isInterrupted()) {
            synchronized (this) {
                if (!running) {
                    return; // stopped while we waited
                }
                closeJmsResources();
                try {
                    openJmsResources();
                    LOG.info("MQ consumer reconnected");
                    return;
                } catch (JMSException e) {
                    LOG.warnf("MQ consumer reconnect failed: %s", e.getLocalizedMessage());
                }
            }
            try {
                Thread.sleep(1000);
            } catch (InterruptedException ignored) {
                Thread.currentThread().interrupt();
                return;
            }
        }
    }

    private void openJmsResources() throws JMSException {
        jmsConnection = connectionFactory.createConnection();
        jmsConnection.setExceptionListener(this::onException);
        jmsSession = jmsConnection.createSession(false, Session.AUTO_ACKNOWLEDGE);
        Queue queue = jmsSession.createQueue(config.queue());
        jmsConsumer = jmsSession.createConsumer(queue);
        jmsConsumer.setMessageListener(this::onMessage);
        jmsConnection.start();
    }

    private void closeJmsResources() {
        closing = true;
        try { if (jmsConsumer   != null) jmsConsumer.close();   } catch (JMSException ignored) {}
        try { if (jmsSession    != null) jmsSession.close();    } catch (JMSException ignored) {}
        try { if (jmsConnection != null) jmsConnection.close(); } catch (JMSException ignored) {}
        jmsConsumer   = null;
        jmsSession    = null;
        jmsConnection = null;
        closing       = false;
    }
}
