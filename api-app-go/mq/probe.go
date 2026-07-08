package mq

import (
	"github.com/ibm-messaging/ibm_mq/api-app-go/config"
)

// ProbeResult holds the resolved connection attributes from a live MQ connection.
type ProbeResult struct {
	Name string // resolved queue manager name
	Host string // resolved host
	Port int    // resolved port
}

// Probe opens a transient MQ connection to verify connectivity and resolve the
// actual queue manager attributes, mirroring InfoResource.java.
// Returns a ProbeResult on success, or an error if the connection fails.
func Probe(cfg *config.Config) (ProbeResult, error) {
	qmgr, err := connect(cfg)
	if err != nil {
		return ProbeResult{}, err
	}
	defer qmgr.Disc()

	// The ibmmq package exposes the queue manager name via qmgr.Name.
	// Host and port resolution is not directly available via the Go client API,
	// so we return the configured values as the resolved values (matching the
	// fallback path in InfoResource.java when WMQ properties are unavailable).
	return ProbeResult{
		Name: qmgr.Name,
		Host: cfg.Host,
		Port: cfg.Port,
	}, nil
}
