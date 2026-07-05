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
import org.jboss.logging.Logger;

import java.util.concurrent.atomic.AtomicLong;

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
    public Response getCount() {
        return Response.ok(new CountResponse(counter.get())).build();
    }

    public record MessageRequest(String text) {}

    public record StatusResponse(String status, String message) {}

    public record CountResponse(long count) {}
}
