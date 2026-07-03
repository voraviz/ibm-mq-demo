package com.example.config;

import io.smallrye.config.ConfigMapping;
import io.smallrye.config.WithName;

@ConfigMapping(prefix = "ibm.mq")
public interface MQConfig {

    String connectionList();

    String host();

    int port();

    String channel();

    @WithName("queue-manager")
    String queueManager();

    String username();

    String password();

    String queue();
}
