package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
	"github.com/ibm-messaging/ibm_mq/api-app-go/mq"
)

// probeTimeout bounds how long /api/info waits for the MQ probe before giving
// up and reporting disconnected. The probe dials each HA node sequentially, so
// an unreachable node could otherwise stall the request thread.
const probeTimeout = 3 * time.Second

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

	// Run the (blocking, non-context-aware) MQ probe in a goroutine and bound
	// it with probeTimeout. The buffered channel lets the goroutine finish and
	// Disc() on its own even if we've already timed out — no leak.
	type probeResult struct {
		r   mq.ProbeResult
		err error
	}
	ch := make(chan probeResult, 1)
	go func() {
		r, err := mq.Probe(h.cfg)
		ch <- probeResult{r, err}
	}()

	select {
	case out := <-ch:
		if out.err == nil {
			resp.Connected = true
			if out.r.Name != "" {
				resp.QueueManager = out.r.Name
			}
			if out.r.Host != "" {
				resp.Host = out.r.Host
			}
			if out.r.Port != 0 {
				resp.Port = out.r.Port
			}
		}
	case <-time.After(probeTimeout):
		log.Printf("MQ info probe timed out after %s — reporting disconnected", probeTimeout)
	}

	writeJSON(w, http.StatusOK, resp)
}
