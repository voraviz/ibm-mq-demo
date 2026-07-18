package com.example.ws;

import io.quarkus.websockets.next.OnClose;
import io.quarkus.websockets.next.OnOpen;
import io.quarkus.websockets.next.OpenConnections;
import io.quarkus.websockets.next.WebSocket;
import io.quarkus.websockets.next.WebSocketConnection;
import jakarta.inject.Inject;
import org.jboss.logging.Logger;

@WebSocket(path = "/ws/messages")
public class MessageWebSocket {

    private static final Logger LOG = Logger.getLogger(MessageWebSocket.class);

    @Inject
    OpenConnections openConnections;

    @OnOpen
    public void onOpen(WebSocketConnection connection) {
        // connection tracking is handled automatically by OpenConnections
    }

    @OnClose
    public void onClose(WebSocketConnection connection) {
        // connection tracking is handled automatically by OpenConnections
    }

    public void broadcast(String message) {
        // Fire-and-forget per connection: never block the caller (the MQ consumer
        // thread), and isolate failures so one dead/slow client can't stop delivery
        // to the others — or, if this threw, kill the consumer thread.
        openConnections.forEach(conn -> conn.sendText(message).subscribe().with(
                ignored -> {},
                err -> LOG.warnf("WebSocket send failed for connection %s: %s",
                        conn.id(), err.getMessage())));
    }
}
