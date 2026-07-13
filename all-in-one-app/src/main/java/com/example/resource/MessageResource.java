package com.example.resource;

import io.micrometer.core.annotation.Counted;
import io.smallrye.reactive.messaging.annotations.Channel;
import io.smallrye.reactive.messaging.annotations.Emitter;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import jakarta.ws.rs.Consumes;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.POST;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;
import org.eclipse.microprofile.openapi.annotations.Operation;
import org.eclipse.microprofile.openapi.annotations.media.Content;
import org.eclipse.microprofile.openapi.annotations.media.Schema;
import org.eclipse.microprofile.openapi.annotations.parameters.RequestBody;
import org.eclipse.microprofile.openapi.annotations.responses.APIResponse;
import org.eclipse.microprofile.openapi.annotations.responses.APIResponses;
import org.eclipse.microprofile.openapi.annotations.tags.Tag;
import org.jboss.logging.Logger;

import java.util.concurrent.atomic.AtomicLong;

@Tag(name = "MQ Messages", description = "Send messages to and query statistics from IBM MQ")
@ApplicationScoped
@Path("/api/messages")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class MessageResource {

    private static final Logger LOG = Logger.getLogger(MessageResource.class);

    @Inject
    @Channel("mq-put")
    Emitter<String> emitter;

    private final AtomicLong counter = new AtomicLong(0);

    @POST
    @Operation(
        summary = "Send a message to IBM MQ",
        description = "Puts a text message onto the configured IBM MQ queue via SmallRye Reactive Messaging."
    )
    @RequestBody(
        description = "Message payload to send to IBM MQ",
        required = true,
        content = @Content(mediaType = MediaType.APPLICATION_JSON,
                           schema = @Schema(implementation = MessageRequest.class))
    )
    @APIResponses({
        @APIResponse(
            responseCode = "202",
            description = "Message accepted and forwarded to IBM MQ",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = StatusResponse.class))
        ),
        @APIResponse(
            responseCode = "400",
            description = "Request body is missing or the text field is blank",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = StatusResponse.class))
        )
    })
    @Counted(value = "mq.messages.put", description = "Number of messages put to IBM MQ")
    public Response putMessage(MessageRequest request) {
        if (request == null || request.text() == null || request.text().isBlank()) {
            return Response.status(Response.Status.BAD_REQUEST)
                    .entity(new StatusResponse("error", "text must not be blank"))
                    .build();
        }
        long n = counter.incrementAndGet();
        String body = "[#" + n + "] " + request.text();
        LOG.infof("PUT message: %s", body);
        emitter.send(body);
        return Response.accepted(new StatusResponse("sent", body)).build();
    }

    @GET
    @Path("/count")
    @Operation(
        summary = "Get sent message count",
        description = "Returns the total number of messages put to IBM MQ via this endpoint since the application started."
    )
    @APIResponses({
        @APIResponse(
            responseCode = "200",
            description = "Sent message count",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = CountResponse.class))
        )
    })
    public Response getCount() {
        return Response.ok(new CountResponse(counter.get())).build();
    }

    public record MessageRequest(
        @Schema(description = "Text content of the message to send to IBM MQ", example = "Hello IBM MQ!")
        String text
    ) {}

    public record StatusResponse(
        @Schema(description = "Outcome status: 'sent' or 'error'", example = "sent")
        String status,

        @Schema(description = "The formatted message body that was enqueued, or an error description", example = "[#1] Hello IBM MQ!")
        String message
    ) {}

    public record CountResponse(
        @Schema(description = "Total number of messages sent to IBM MQ since startup", example = "7")
        long count
    ) {}
}
