package com.example.config;

import io.smallrye.config.ConfigMapping;
import io.smallrye.config.WithDefault;
import io.smallrye.config.WithName;

import java.util.Optional;

@ConfigMapping(prefix = "ibm.mq")
public interface MQConfig {

    @WithName("ccdt-url")
    Optional<String> ccdtUrl();

    @WithDefault("")
    Optional<String> connectionList();

    @WithName("client-reconnect-timeout")
    int clientReconnectTimeout();

    // Seconds the consumer sleeps between receiveNoWait() pulses when the queue
    // is empty (see application.properties). Must exceed the QM's BALTIMEOUT
    // (balance timeout, default 10s) so the instance stays MOVABLE long enough
    // to rebalance; 15 ~= 1.5x BALTIMEOUT balances reliability vs. idle latency.
    @WithName("consumer-pulse-interval")
    @WithDefault("15")
    int consumerPulseInterval();

    @WithDefault("")
    String channel();

    String applicationName();

    @WithName("queue-manager")
    String queueManager();

    String username();

    String password();

    String queue();
}
