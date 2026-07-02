package com.example.resource;

import com.example.config.MQConfig;
import jakarta.inject.Inject;
import jakarta.jms.Connection;
import jakarta.jms.ConnectionFactory;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;

@Path("/api/info")
@Produces(MediaType.APPLICATION_JSON)
public class InfoResource {

    @Inject
    MQConfig mqConfig;

    @Inject
    ConnectionFactory connectionFactory;

    @GET
    public Response info() {
        boolean connected = false;
        try (Connection c = connectionFactory.createConnection()) {
            connected = true;
        } catch (Exception ignored) {
        }
        return Response.ok(new InfoResponse(mqConfig.queueManager(), mqConfig.host(), mqConfig.port(), connected)).build();
    }

    public record InfoResponse(String queueManager, String host, int port, boolean connected) {}
}
