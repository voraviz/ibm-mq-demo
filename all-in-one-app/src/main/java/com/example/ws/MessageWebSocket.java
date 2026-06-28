package com.example.ws;

import io.quarkus.websockets.next.OnClose;
import io.quarkus.websockets.next.OnOpen;
import io.quarkus.websockets.next.OpenConnections;
import io.quarkus.websockets.next.WebSocket;
import io.quarkus.websockets.next.WebSocketConnection;
import jakarta.inject.Inject;

@WebSocket(path = "/ws/messages")
public class MessageWebSocket {

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
        openConnections.forEach(conn -> conn.sendText(message).subscribeAsCompletionStage().join());
    }
}
