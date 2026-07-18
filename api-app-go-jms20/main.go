package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// ── HTTP server with timeouts ───────────────────────────────────────────
	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           middleware.CORS(mux),
		ReadHeaderTimeout: 10 * time.Second,  // slowloris guard
		IdleTimeout:       120 * time.Second, // reap idle keep-alive conns
		// No WriteTimeout — it would kill the long-lived /ws/messages stream.
	}

	// Run the server in the background so main can wait for a shutdown signal.
	go func() {
		log.Printf("Starting IBM MQ JMS20 API on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM: stop accepting requests and drain
	// in-flight ones, then disconnect the MQ consumer cleanly.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Shutdown signal received — draining")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	consumer.Stop()
	log.Println("Shutdown complete")
}
