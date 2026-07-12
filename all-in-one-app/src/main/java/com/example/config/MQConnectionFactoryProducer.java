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

    @Produces
    @ApplicationScoped
    public ConnectionFactory connectionFactory() throws JMSException {
        MQConnectionFactory factory = new MQConnectionFactory();

        String ccdtUrl = config.ccdtUrl().filter(s -> !s.isBlank()).orElse(null);
        if (ccdtUrl != null) {
            factory.setStringProperty(WMQConstants.WMQ_CCDTURL, ccdtUrl);
            LOG.info("Use CCDT: " + ccdtUrl);
            LOG.debug("CCDT URL set — skipping connectionList and channel");
        } else {
            factory.setConnectionNameList(config.connectionList());
            factory.setChannel(config.channel());
            LOG.info("Use Connection List: " + config.connectionList());
            LOG.debug("Connection list: " + config.connectionList() + ", channel: " + config.channel());
        }

        factory.setQueueManager(config.queueManager());
        // factory.setTransportType(WMQConstants.WMQ_CM_CLIENT);
        factory.setIntProperty(WMQConstants.WMQ_CONNECTION_MODE, WMQConstants.WMQ_CM_CLIENT);
        factory.setStringProperty(WMQConstants.USERID, config.username());
        factory.setStringProperty(WMQConstants.PASSWORD, config.password());
        factory.setStringProperty(WMQConstants.WMQ_APPLICATIONNAME, config.applicationName());
        LOG.debug("MQ factory configured — queueManager: " + config.queueManager()
                + ", applicationName: " + config.applicationName()
                + ", reconnectOptions: WMQ_CLIENT_RECONNECT, reconnectTimeout: 30s");
        // Automatically reconnect to the queue manager when it becomes available again.
        // WMQ_CLIENT_RECONNECT retries indefinitely until the QM is reachable.
        // WMQ_CLIENT_RECONNECT_TIMEOUT (seconds) caps how long a single reconnect
        // attempt waits before the client gives up and throws to the application.
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_TIMEOUT, 30); // 30 sec
        return factory;
    }
}
