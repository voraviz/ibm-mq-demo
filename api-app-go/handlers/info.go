package handlers

import (
	"net/http"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/ibm_mq/api-app-go/mq"
)

// InfoHandler handles GET /api/info.
type InfoHandler struct {
	cfg *config.Config
}

// NewInfoHandler creates an InfoHandler.
func NewInfoHandler(cfg *config.Config) *InfoHandler {
	return &InfoHandler{cfg: cfg}
}

type infoResponse struct {
	QueueManager string `json:"queueManager"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Connected    bool   `json:"connected"`
}

// GetInfo handles GET /api/info.
// It attempts a live MQ connection to verify connectivity and resolve the
// actual queue manager name and host, mirroring InfoResource.java.
func (h *InfoHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	resp := infoResponse{
		QueueManager: h.cfg.QueueManager,
		Host:         h.cfg.Host,
		Port:         h.cfg.Port,
		Connected:    false,
	}

	qmgr, err := mq.Probe(h.cfg)
	if err == nil {
		resp.Connected = true
		if qmgr.Name != "" {
			resp.QueueManager = qmgr.Name
		}
		if qmgr.Host != "" {
			resp.Host = qmgr.Host
		}
		if qmgr.Port != 0 {
			resp.Port = qmgr.Port
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
