package com.example.messaging;

import com.example.config.MQConfig;
import com.example.ws.MessageWebSocket;
import jakarta.annotation.PreDestroy;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import jakarta.jms.Connection;
import jakarta.jms.ConnectionFactory;
import jakarta.jms.JMSException;
import jakarta.jms.MessageConsumer;
import jakarta.jms.Queue;
import jakarta.jms.Session;
import jakarta.jms.TextMessage;
import org.jboss.logging.Logger;

@ApplicationScoped
public class MQConsumer {

    private static final Logger LOG = Logger.getLogger(MQConsumer.class);

    @Inject
    ConnectionFactory connectionFactory;

    @Inject
    MQConfig config;

    @Inject
    MessageWebSocket webSocket;

    // JMS resources — null when stopped
    private volatile Connection      jmsConnection;
    private volatile Session         jmsSession;
    private volatile MessageConsumer jmsConsumer;
    private volatile Thread          listenerThread;
    private volatile boolean         closing = false;

    public synchronized void start() {
        if (jmsConnection != null) {
            return; // already running
        }
        try {
            openJmsResources();
            listenerThread = Thread.ofPlatform().name("mq-consumer").daemon(true).start(this::readLoop);
            LOG.info("MQ consumer started — listening on queue: " + config.queue());
        } catch (JMSException e) {
            LOG.error("Failed to start MQ consumer", e);
            closeJmsResources();
        }
    }

    public synchronized void stop() {
        if (jmsConnection == null) {
            return; // already stopped
        }
        closeJmsResources();
        LOG.info("MQ consumer stopped");
    }

    public synchronized boolean isRunning() {
        return jmsConnection != null;
    }

    @PreDestroy
    public synchronized void onDestroy() {
        if (jmsConnection != null) {
            closeJmsResources();
            LOG.info("MQ consumer shut down on application stop");
        }
    }

    // ── private ──────────────────────────────────────────────────────────────

    private void readLoop() {
        while (!Thread.currentThread().isInterrupted()) {
            try {
                jakarta.jms.Message msg = jmsConsumer.receive(500);
                if (msg == null) {
                    continue; // timeout — check interrupt and loop
                }
                if (msg instanceof TextMessage textMsg) {
                    String text = textMsg.getText();
                    LOG.infof("GET message: %s", text);
                    webSocket.broadcast(text);
                }
            } catch (JMSException e) {
                if (closing) {
                    return;
                }
                LOG.warnf("MQ consumer disconnected unexpectedly: %s — reconnecting", e.getLocalizedMessage());
                reconnectLoop();
            }
        }
    }

    private void reconnectLoop() {
        while (!closing && !Thread.currentThread().isInterrupted()) {
            synchronized (this) {
                closeJmsResources(false);
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
        jmsSession = jmsConnection.createSession(false, Session.AUTO_ACKNOWLEDGE);
        Queue queue = jmsSession.createQueue(config.queue());
        jmsConsumer = jmsSession.createConsumer(queue);
        jmsConnection.start();
    }

    private void closeJmsResources() {
        closeJmsResources(true);
    }

    private void closeJmsResources(boolean interruptListener) {
        closing = true;
        if (interruptListener && listenerThread != null) {
            listenerThread.interrupt();
            listenerThread = null;
        }
        try { if (jmsConsumer   != null) jmsConsumer.close();   } catch (JMSException ignored) {}
        try { if (jmsSession    != null) jmsSession.close();    } catch (JMSException ignored) {}
        try { if (jmsConnection != null) jmsConnection.close(); } catch (JMSException ignored) {}
        jmsConsumer   = null;
        jmsSession    = null;
        jmsConnection = null;
        closing       = false;
    }
}
