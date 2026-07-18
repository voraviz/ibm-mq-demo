package com.example.resource;

import com.example.messaging.MQConsumer;
import jakarta.inject.Inject;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.POST;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;
import org.eclipse.microprofile.openapi.annotations.Operation;
import org.eclipse.microprofile.openapi.annotations.media.Content;
import org.eclipse.microprofile.openapi.annotations.media.Schema;
import org.eclipse.microprofile.openapi.annotations.responses.APIResponse;
import org.eclipse.microprofile.openapi.annotations.responses.APIResponses;
import org.eclipse.microprofile.openapi.annotations.tags.Tag;

@Tag(name = "MQ Consumer", description = "Control and monitor the IBM MQ message consumer")
@Path("/api/consumer")
@Produces(MediaType.APPLICATION_JSON)
public class ConsumerResource {

    @Inject
    MQConsumer consumer;

    @POST
    @Path("/start")
    @Operation(
        summary = "Start the MQ consumer",
        description = "Starts the background IBM MQ consumer thread if it is not already running."
    )
    @APIResponses({
        @APIResponse(
            responseCode = "200",
            description = "Consumer started successfully",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = StatusResponse.class))
        )
    })
    public Response start() {
        consumer.start();
        return Response.ok(new StatusResponse("started")).build();
    }

    @POST
    @Path("/stop")
    @Operation(
        summary = "Stop the MQ consumer",
        description = "Stops the background IBM MQ consumer thread if it is currently running."
    )
    @APIResponses({
        @APIResponse(
            responseCode = "200",
            description = "Consumer stopped successfully",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = StatusResponse.class))
        )
    })
    public Response stop() {
        consumer.stop();
        return Response.ok(new StatusResponse("stopped")).build();
    }

    @GET
    @Path("/status")
    @Operation(
        summary = "Get consumer status",
        description = "Returns whether the IBM MQ consumer is currently running or stopped."
    )
    @APIResponses({
        @APIResponse(
            responseCode = "200",
            description = "Current consumer status",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = StatusResponse.class))
        )
    })
    public Response status() {
        return Response.ok(new StatusResponse(consumer.isRunning() ? "running" : "stopped")).build();
    }

    @GET
    @Path("/count")
    @Operation(
        summary = "Get consumed message count",
        description = "Returns the total number of messages consumed from IBM MQ since the application started."
    )
    @APIResponses({
        @APIResponse(
            responseCode = "200",
            description = "Consumed message count",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = CountResponse.class))
        )
    })
    public Response count() {
        return Response.ok(new CountResponse(consumer.getCount())).build();
    }

    public record StatusResponse(
        @Schema(description = "Consumer status value: 'running', 'started', or 'stopped'", examples = {"running"})
        String status
    ) {}

    public record CountResponse(
        @Schema(description = "Total number of messages consumed from IBM MQ", examples = {"42"})
        long count
    ) {}
}
