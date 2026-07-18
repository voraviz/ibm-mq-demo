package com.example.resource;

import com.example.config.MQConfig;
import com.example.config.ProbeConnection;
import com.ibm.msg.client.jakarta.jms.JmsConnection;
import com.ibm.msg.client.jakarta.wmq.WMQConstants;
import jakarta.inject.Inject;
import jakarta.jms.Connection;
import jakarta.jms.ConnectionFactory;
import jakarta.ws.rs.GET;
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
import org.jboss.logging.Logger;

import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

@Tag(name = "MQ Info", description = "IBM MQ connection information")
@Path("/api/info")
@Produces(MediaType.APPLICATION_JSON)
public class InfoResource {

    private static final Logger LOG = Logger.getLogger(InfoResource.class);

    // How long /info will wait for the connection probe before giving up and
    // reporting disconnected. Bounds request latency so a slow/unreachable MQ
    // server can't tie up the request thread.
    private static final long PROBE_TIMEOUT_SECONDS = 3;

    @Inject
    MQConfig mqConfig;

    @Inject
    @ProbeConnection
    ConnectionFactory connectionFactory;

    @GET
    @Operation(
        summary = "Get MQ connection info",
        description = "Returns the queue manager name, host, port, and whether a live connection to IBM MQ can be established."
    )
    @APIResponses({
        @APIResponse(
            responseCode = "200",
            description = "MQ connection information retrieved successfully",
            content = @Content(mediaType = MediaType.APPLICATION_JSON,
                               schema = @Schema(implementation = InfoResponse.class))
        ),
        @APIResponse(
            responseCode = "500",
            description = "Unexpected server error"
        )
    })
    public Response info() {
        String queueManager = mqConfig.queueManager();

        LOG.debug("Requesting MQ info — configured queueManager: " + queueManager);

        InfoResponse probe;
        try {
            probe = CompletableFuture.supplyAsync(() -> this.probe(queueManager))
                    .get(PROBE_TIMEOUT_SECONDS, TimeUnit.SECONDS);
        } catch (TimeoutException e) {
            LOG.warnf("MQ info probe timed out after %ds — reporting disconnected", PROBE_TIMEOUT_SECONDS);
            probe = new InfoResponse(queueManager, "", 0, false);
        } catch (Exception e) {
            LOG.warn("MQ info probe failed: " + e.getMessage());
            probe = new InfoResponse(queueManager, "", 0, false);
        }

        LOG.debug("InfoResponse: connected=" + probe.connected() + ", queueManager=" + probe.queueManager()
                + ", host=" + probe.host() + ", port=" + probe.port());
        return Response.ok(probe).build();
    }

    // Opens a short-lived connection to resolve live queue manager/host/port.
    // Always closes the connection it opens (even if /info already timed out and
    // abandoned the result), so an eventually-successful connect can't leak.
    private InfoResponse probe(String configuredQueueManager) {
        String queueManager = configuredQueueManager;
        String host = "";
        int port = 0;

        try (Connection connection = connectionFactory.createConnection()) {
            LOG.debug("MQ connection established successfully");

            if (connection instanceof JmsConnection jmsConnection) {
                String resolvedQueueManager = jmsConnection.getStringProperty(WMQConstants.WMQ_RESOLVED_QUEUE_MANAGER);
                String resolvedHost = jmsConnection.getStringProperty(WMQConstants.WMQ_HOST_NAME);
                int resolvedPort = jmsConnection.getIntProperty(WMQConstants.WMQ_PORT);
                if (resolvedQueueManager != null && !resolvedQueueManager.isBlank()) {
                    queueManager = resolvedQueueManager;
                }
                if (resolvedHost != null && !resolvedHost.isBlank()) {
                    host = resolvedHost;
                }
                port = resolvedPort;
                LOG.debug("MQ Server Host: " + host + " (" + port + "), resolvedQueueManager: " + queueManager);
            }
            return new InfoResponse(queueManager, host, port, true);
        } catch (Exception e) {
            LOG.warn("MQ connection failed: " + e.getMessage());
            return new InfoResponse(queueManager, "", 0, false);
        }
    }

    public record InfoResponse(
        @Schema(description = "Name of the connected queue manager", example = "QM1")
        String queueManager,

        @Schema(description = "Hostname or IP address of the IBM MQ server", example = "localhost")
        String host,

        @Schema(description = "Port number of the IBM MQ listener", example = "1414")
        int port,

        @Schema(description = "Whether a live connection to IBM MQ was established successfully")
        boolean connected
    ) {}
}
