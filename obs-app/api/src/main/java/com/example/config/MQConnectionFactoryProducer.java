package com.example.config;

import com.ibm.mq.jakarta.jms.MQConnectionFactory;
import com.ibm.msg.client.jakarta.wmq.WMQConstants;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.enterprise.inject.Produces;
import jakarta.inject.Inject;
import jakarta.jms.ConnectionFactory;
import jakarta.jms.JMSException;

@ApplicationScoped
public class MQConnectionFactoryProducer {

    @Inject
    MQConfig config;

    @Produces
    @ApplicationScoped
    public ConnectionFactory connectionFactory() throws JMSException {
        MQConnectionFactory factory = new MQConnectionFactory();
        factory.setConnectionNameList(config.connectionList());
        factory.setChannel(config.channel());
        factory.setQueueManager(config.queueManager());
        factory.setTransportType(WMQConstants.WMQ_CM_CLIENT);
        factory.setStringProperty(WMQConstants.USERID, config.username());
        factory.setStringProperty(WMQConstants.PASSWORD, config.password());
        // Automatically reconnect to the queue manager when it becomes available again.
        // WMQ_CLIENT_RECONNECT retries indefinitely until the QM is reachable.
        // WMQ_CLIENT_RECONNECT_TIMEOUT (seconds) caps how long a single reconnect
        // attempt waits before the client gives up and throws to the application.
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_OPTIONS, WMQConstants.WMQ_CLIENT_RECONNECT);
        factory.setIntProperty(WMQConstants.WMQ_CLIENT_RECONNECT_TIMEOUT, 30); // 30 sec
        return factory;
    }
}
