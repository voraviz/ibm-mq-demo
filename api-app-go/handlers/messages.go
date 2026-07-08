package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ibm-messaging/ibm_mq/api-app-go/mq"
	"github.com/prometheus/client_golang/prometheus"
)

// MessagesHandler handles POST /api/messages and GET /api/messages/count.
type MessagesHandler struct {
	producer *mq.Producer
	counter  prometheus.Counter
}

// NewMessagesHandler creates a MessagesHandler and registers the Prometheus counter.
func NewMessagesHandler(producer *mq.Producer, reg prometheus.Registerer) *MessagesHandler {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mq_messages_put_total",
		Help: "Number of messages put to IBM MQ",
	})
	reg.MustRegister(counter)
	return &MessagesHandler{producer: producer, counter: counter}
}

type messageRequest struct {
	Text string `json:"text"`
}

type statusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type countResponse struct {
	Count int64 `json:"count"`
}

// PutMessage handles POST /api/messages.
func (h *MessagesHandler) PutMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var req messageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Text) == "" {
		writeJSON(w, http.StatusBadRequest, statusResponse{Status: "error", Message: "text must not be blank"})
		return
	}

	body, err := h.producer.Put(req.Text)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, statusResponse{Status: "error", Message: err.Error()})
		return
	}

	h.counter.Inc()
	writeJSON(w, http.StatusAccepted, statusResponse{Status: "sent", Message: body})
}

// GetCount handles GET /api/messages/count.
func (h *MessagesHandler) GetCount(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, countResponse{Count: h.producer.PutCount()})
}
