package mq

import (
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

// resolveActiveNode iterates the parsed entries from cfg.ConnectionList and
// returns the first one that accepts a CreateContext() (i.e. the active HA node).
// Falls back to cfg.Host/cfg.Port if no individual probe succeeds.
func resolveActiveNode(cfg *config.Config) (string, int) {
	entries := parseConnEntries(connectionName(cfg))
	if len(entries) == 1 {
		return entries[0].host, entries[0].port
	}
	for _, e := range entries {
		// Build a single-host config for this entry (ConnectionList cleared so
		// newConnectionFactory uses the Host/Port fallback path).
		singleCfg := *cfg
		singleCfg.Host = e.host
		singleCfg.Port = e.port
		singleCfg.ConnectionList = ""

		singleCF := newConnectionFactory(&singleCfg)
		ctx, jmsErr := singleCF.CreateContext(multiHostOption(&singleCfg))
		if jmsErr != nil {
			continue
		}
		ctx.Close()
		return e.host, e.port
	}
	// Defensive fallback — should not be reached since the initial Probe()
	// CreateContext() already succeeded.
	return cfg.Host, cfg.Port
}

// Probe opens a transient JMS connection to verify connectivity and resolve the
// actual queue manager name and the active HA node host/port.
// Returns a ProbeResult on success, or an error if the connection fails.
func Probe(cfg *config.Config) (ProbeResult, error) {
	cf := newConnectionFactory(cfg)
	ctx, jmsErr := cf.CreateContext(multiHostOption(cfg))
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
