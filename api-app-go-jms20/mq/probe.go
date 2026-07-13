package mq

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/ibm-messaging/ibm_mq/api-app-go-jms20/config"
)

// ProbeResult holds the resolved connection attributes from a live MQ connection.
type ProbeResult struct {
	Name string // resolved queue manager name
	Host string // resolved host of the active HA node
	Port int    // resolved port of the active HA node
}

// connEntry is a parsed host(port) token from a ConnectionName list.
type connEntry struct {
	host string
	port int
}

// parseConnEntries splits an MQ ConnectionName string into individual entries.
// Input format: "host1(port1),host2(port2),..." or "host(port)".
// Entries that cannot be parsed are silently skipped.
func parseConnEntries(s string) []connEntry {
	var entries []connEntry
	for _, token := range strings.Split(s, ",") {
		token = strings.TrimSpace(token)
		open := strings.LastIndex(token, "(")
		close := strings.LastIndex(token, ")")
		if open < 0 || close <= open {
			continue
		}
		host := strings.TrimSpace(token[:open])
		port, err := strconv.Atoi(strings.TrimSpace(token[open+1 : close]))
		if err != nil || host == "" {
			continue
		}
		entries = append(entries, connEntry{host: host, port: port})
	}
	return entries
}

// parseCcdtEntries reads a file-based CCDT JSON and returns all host/port
// pairs found across all clientConnection entries.
// Returns nil for HTTP/HTTPS CCDTs (cannot be read locally).
func parseCcdtEntries(ccdtUrl string) []connEntry {
	// Only handle file:// CCDTs — HTTP CCDTs are remote and unreadable here.
	path := ""
	switch {
	case strings.HasPrefix(ccdtUrl, "file:///"):
		path = ccdtUrl[len("file://"):]
	case strings.HasPrefix(ccdtUrl, "file://"):
		// file://host/path form — unsupported, skip
		return nil
	case strings.HasPrefix(ccdtUrl, "file:"):
		path = ccdtUrl[len("file:"):]
	case strings.HasPrefix(ccdtUrl, "http://"), strings.HasPrefix(ccdtUrl, "https://"):
		return nil
	default:
		path = ccdtUrl
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var ccdt struct {
		Channel []struct {
			ClientConnection struct {
				Connection []struct {
					Host string `json:"host"`
					Port int    `json:"port"`
				} `json:"connection"`
			} `json:"clientConnection"`
		} `json:"channel"`
	}
	if err := json.Unmarshal(data, &ccdt); err != nil {
		return nil
	}

	var entries []connEntry
	for _, ch := range ccdt.Channel {
		for _, conn := range ch.ClientConnection.Connection {
			if conn.Host != "" && conn.Port > 0 {
				entries = append(entries, connEntry{host: conn.Host, port: conn.Port})
			}
		}
	}
	return entries
}

// resolveActiveNode probes each host/port entry and returns the first one that
// accepts a direct (non-reconnecting) CreateContext() — i.e. the active HA/cluster node.
// Without MQCNO_RECONNECT a standby immediately returns MQRC_STANDBY_Q_MGR
// (2539) instead of silently redirecting the probe to the active node.
//
// For CCDT mode the host list is extracted from the CCDT JSON file. If the
// CCDT is remote (HTTP) or unreadable, returns ("", 0).
func resolveActiveNode(cfg *config.Config) (string, int) {
	var entries []connEntry
	if cfg.CcdtUrl != "" {
		entries = parseCcdtEntries(cfg.CcdtUrl)
	} else {
		entries = parseConnEntries(connectionName(cfg))
	}

	if len(entries) == 1 {
		return entries[0].host, entries[0].port
	}
	for _, e := range entries {
		// Build a single-host config for this entry (ConnectionList/CcdtUrl
		// cleared so newConnectionFactory uses the Host/Port fallback path).
		singleCfg := *cfg
		singleCfg.Host = e.host
		singleCfg.Port = e.port
		singleCfg.ConnectionList = ""
		singleCfg.CcdtUrl = ""

		singleCF := newConnectionFactory(&singleCfg)
		ctx, jmsErr := singleCF.CreateContext(connectOptionNoReconnect(&singleCfg))
		if jmsErr != nil {
			continue
		}
		ctx.Close()
		return e.host, e.port
	}
	// Fallback: CCDT unreadable (HTTP) or all probes failed.
	return "", 0
}

// Probe opens a transient JMS connection to verify connectivity and resolve the
// actual queue manager name and the active HA node host/port.
// Returns a ProbeResult on success, or an error if the connection fails.
func Probe(cfg *config.Config) (ProbeResult, error) {
	cf := newConnectionFactory(cfg)
	ctx, jmsErr := cf.CreateContext(connectOption(cfg))
	if jmsErr != nil {
		return ProbeResult{}, jmsErr
	}
	defer ctx.Close()

	host, port := resolveActiveNode(cfg)

	return ProbeResult{
		Name: cfg.QueueManager,
		Host: host,
		Port: port,
	}, nil
}
