package com.example.resource;

import com.example.messaging.MQConsumer;
import jakarta.inject.Inject;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.POST;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;

@Path("/api/consumer")
@Produces(MediaType.APPLICATION_JSON)
public class ConsumerResource {

    @Inject
    MQConsumer consumer;

    @POST
    @Path("/start")
    public Response start() {
        consumer.start();
        return Response.ok(new StatusResponse("started")).build();
    }

    @POST
    @Path("/stop")
    public Response stop() {
        consumer.stop();
        return Response.ok(new StatusResponse("stopped")).build();
    }

    @GET
    @Path("/status")
    public Response status() {
        return Response.ok(new StatusResponse(consumer.isRunning() ? "running" : "stopped")).build();
    }

    @GET
    @Path("/count")
    public Response count() {
        return Response.ok(new CountResponse(consumer.getCount())).build();
    }

    public record StatusResponse(String status) {}

    public record CountResponse(long count) {}
}
