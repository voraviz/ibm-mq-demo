package com.example.config;

import com.ibm.mq.jakarta.jms.MQConnectionFactory;
import com.ibm.msg.client.jakarta.wmq.WMQConstants;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.enterprise.inject.Produces;
import jakarta.inject.Inject;
import jakarta.jms.ConnectionFactory;
import jakarta.jms.JMSException;
import org.jboss.logging.Logger;

@ApplicationScoped
public class MQConnectionFactoryProducer {
    private static final Logger LOG = Logger.getLogger(MQConnectionFactoryProducer.class);

    @Inject
    MQConfig config;

    /**
     * Default factory used by the consumer and outbound producer. Client
     * auto-reconnect is enabled so long-lived connections survive queue manager
     * outages transparently.
     */
    @Produces
    @ApplicationScoped
    public ConnectionFactory connectionFactory() throws JMSException {
        MQConnectionFactory factory = new MQConnectionFactory();
        applyConnectionConfig(factory);
        // Automatically reconnect to the queue manager when it becomes available again.
        // WMQ_CLIENT_RECONNECT retries across all hosts until the QM is reachable.
        // WMQ_CLIENT_RECONNECT_TIMEOUT (seconds) is the TOTAL reconnect window — how
        // long the client keeps retrying before it gives up and throws to the app
        // (ibm.mq.client-reconnect-timeout, 900 s here; IBM default 1800 s).
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_TIMEOUT, config.clientReconnectTimeout()); 
        LOG.debug("MQ factory configured — queueManager: " + config.queueManager()
                + ", applicationName: " + config.applicationName()
                + ", reconnectOptions: WMQ_CLIENT_RECONNECT + reconnectTimeout: "+ config.clientReconnectTimeout()+"s");

        return factory;
    }

    /**
     * Factory dedicated to one-shot connectivity probes (e.g. {@code /api/info}).
     * Client reconnect is DISABLED: a probe reports the current state and must
     * fail fast rather than block retrying a down queue manager.
     */
    @Produces
    @ApplicationScoped
    @ProbeConnection
    public ConnectionFactory probeConnectionFactory() throws JMSException {
        MQConnectionFactory factory = new MQConnectionFactory();
        applyConnectionConfig(factory);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT_DISABLED);
        // Distinct app name so probe connections (reconnect DISABLED → always
        // IMMREASN(NOTRECONN)) don't pollute the workload's APSTATUS balancing view.
        // WMQ_APPLICATIONNAME / APPLTAG caps at 28 chars — keep the base name short.
        String probeName = config.applicationName() + "-probe";
        factory.setStringProperty(WMQConstants.WMQ_APPLICATIONNAME, probeName);
        LOG.debug("MQ probe factory configured — queueManager: " + config.queueManager()
                + ", applicationName: " + probeName
                + ", reconnectOptions: WMQ_CLIENT_RECONNECT_DISABLED");

        return factory;
    }

    /**
     * Applies the connection target (CCDT or connection list), queue manager and
     * credentials that are common to every factory. Reconnect options are left to
     * the caller so each factory can choose its own policy.
     */
    private void applyConnectionConfig(MQConnectionFactory factory) throws JMSException {
        // Set client mode before the CCDT. Without this, the MQ client can
        // validate CCDTURL as though this were a bindings-mode connection.
        factory.setTransportType(WMQConstants.WMQ_CM_CLIENT);

        String ccdtUrl = config.ccdtUrl().filter(s -> !s.isBlank()).orElse(null);
        if (ccdtUrl != null) {
            factory.setStringProperty(WMQConstants.WMQ_CCDTURL, ccdtUrl);
            LOG.info("Use CCDT: " + ccdtUrl);
            LOG.debug("CCDT URL set — skipping connectionList and channel");
        } else {
            String connectionList = config.connectionList().filter(s -> !s.isBlank()).orElse(null);
            factory.setConnectionNameList(connectionList);
            factory.setChannel(config.channel());
            LOG.info("Use Connection List: " + connectionList);
            LOG.debug("Connection list: " + connectionList + ", channel: " + config.channel());
        }

        factory.setQueueManager(config.queueManager());
        factory.setStringProperty(WMQConstants.USERID, config.username());
        factory.setStringProperty(WMQConstants.PASSWORD, config.password());
        factory.setStringProperty(WMQConstants.WMQ_APPLICATIONNAME, config.applicationName());
        // Uniform-cluster app balancing timeout (BALTIMEOUT): how long MQ waits for
        // the app to reach a transaction/delivery boundary before forcing a rebalance.
        // Optional — when unset, MQ uses its own default.
        if (config.balancingTimeout().isPresent()) {
            factory.setBalancingTimeout(config.balancingTimeout().getAsInt());
        }
        LOG.debug("BALTIMEOUT: " + factory.getBalancingTimeout() + "s");
    }
}
