package com.example.config;

import io.smallrye.config.ConfigMapping;
import io.smallrye.config.WithDefault;
import io.smallrye.config.WithName;

@ConfigMapping(prefix = "ibm.mq")
public interface MQConfig {

    @WithName("ccdt-url")
    @WithDefault("")
    String ccdtUrl();

    @WithDefault("")
    String connectionList();

    String host();

    int port();

    @WithDefault("")
    String channel();

    String applicationName();

    @WithName("queue-manager")
    String queueManager();

    String username();

    String password();

    String queue();
}
