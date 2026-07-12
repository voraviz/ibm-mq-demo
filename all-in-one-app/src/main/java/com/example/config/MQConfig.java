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
    String connectionList();

    @WithDefault("")
    String channel();

    String applicationName();

    @WithName("queue-manager")
    String queueManager();

    String username();

    String password();

    String queue();
}
