package handlers

import (
	"net/http"

	"github.com/ibm-messaging/ibm_mq/api-app-go/mq"
)

// ConsumerHandler handles all /api/consumer/* endpoints.
type ConsumerHandler struct {
	consumer *mq.Consumer
}

// NewConsumerHandler creates a ConsumerHandler.
func NewConsumerHandler(consumer *mq.Consumer) *ConsumerHandler {
	return &ConsumerHandler{consumer: consumer}
}

type consumerStatusResponse struct {
	Status string `json:"status"`
}

type consumerCountResponse struct {
	Count int64 `json:"count"`
}

// Start handles POST /api/consumer/start.
func (h *ConsumerHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.consumer.Start()
	writeJSON(w, http.StatusOK, consumerStatusResponse{Status: "started"})
}

// Stop handles POST /api/consumer/stop.
func (h *ConsumerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	h.consumer.Stop()
	writeJSON(w, http.StatusOK, consumerStatusResponse{Status: "stopped"})
}

// Status handles GET /api/consumer/status.
func (h *ConsumerHandler) Status(w http.ResponseWriter, r *http.Request) {
	status := "stopped"
	if h.consumer.IsRunning() {
		status = "running"
	}
	writeJSON(w, http.StatusOK, consumerStatusResponse{Status: status})
}

// Count handles GET /api/consumer/count.
func (h *ConsumerHandler) Count(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, consumerCountResponse{Count: h.consumer.GetCount()})
}
