package com.example.config;

import io.smallrye.config.ConfigMapping;
import io.smallrye.config.WithDefault;
import io.smallrye.config.WithName;

import java.util.Optional;
import java.util.OptionalInt;

@ConfigMapping(prefix = "ibm.mq")
public interface MQConfig {

    @WithName("ccdt-url")
    Optional<String> ccdtUrl();

    @WithDefault("")
    Optional<String> connectionList();

    @WithName("client-reconnect-timeout")
    int clientReconnectTimeout();

    @WithName("balancing-timeout")
    OptionalInt balancingTimeout();

    @WithDefault("")
    String channel();

    String applicationName();

    @WithName("queue-manager")
    String queueManager();

    String username();

    String password();

    String queue();
}
