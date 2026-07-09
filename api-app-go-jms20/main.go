package main

import (
	"log"
	"net/http"

	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/config"
	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/handlers"
	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/middleware"
	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/mq"
	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/ws"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()

	// ── IBM MQ components ──────────────────────────────────────────────────
	producer := mq.NewProducer(cfg)
	consumer := mq.NewConsumer(cfg)

	// ── WebSocket hub ──────────────────────────────────────────────────────
	hub := ws.NewHub()

	// Wire the consumer to broadcast received messages to all WS clients
	consumer.SetBroadcaster(hub.Broadcast)

	// ── Prometheus registry ────────────────────────────────────────────────
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// ── HTTP handlers ──────────────────────────────────────────────────────
	messagesH := handlers.NewMessagesHandler(producer, reg)
	consumerH := handlers.NewConsumerHandler(consumer)
	infoH := handlers.NewInfoHandler(cfg)

	// ── Routes ─────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Messages
	mux.HandleFunc("/api/messages", messagesH.PutMessage)
	mux.HandleFunc("/api/messages/count", messagesH.GetCount)

	// Consumer control
	mux.HandleFunc("/api/consumer/start", consumerH.Start)
	mux.HandleFunc("/api/consumer/stop", consumerH.Stop)
	mux.HandleFunc("/api/consumer/status", consumerH.Status)
	mux.HandleFunc("/api/consumer/count", consumerH.Count)

	// Info
	mux.HandleFunc("/api/info", infoH.GetInfo)

	// WebSocket
	mux.HandleFunc("/ws/messages", hub.ServeWS)

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// ── Apply CORS middleware ───────────────────────────────────────────────
	addr := ":" + cfg.ServerPort
	log.Printf("Starting IBM MQ JMS20 API on %s", addr)
	if err := http.ListenAndServe(addr, middleware.CORS(mux)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
