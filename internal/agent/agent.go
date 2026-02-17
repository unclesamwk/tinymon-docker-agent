package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/unclesamwk/tinymon-docker-agent/internal/docker"
	"github.com/unclesamwk/tinymon-docker-agent/internal/tinymon"
)

type Agent struct {
	tm        *tinymon.Client
	agentName string
	known     map[string]bool
}

func New(tm *tinymon.Client, agentName string) *Agent {
	return &Agent{
		tm:        tm,
		agentName: agentName,
		known:     make(map[string]bool),
	}
}

func (a *Agent) Sync(ctx context.Context) {
	containers, err := docker.ListEnabled(ctx)
	if err != nil {
		log.Printf("ERROR listing containers: %v", err)
		return
	}

	seen := make(map[string]bool)
	var results []tinymon.Result

	for _, c := range containers {
		addr := a.address(c.Name)
		seen[addr] = true

		topic := c.Topic
		if topic == "" {
			topic = "Docker/" + a.agentName
		}

		host := tinymon.Host{
			Name:        c.DisplayName,
			Address:     addr,
			Description: fmt.Sprintf("Container %s", c.Name),
			Topic:       topic,
			Enabled:     1,
		}
		if err := a.tm.UpsertHost(host); err != nil {
			log.Printf("ERROR upsert host %s: %v", addr, err)
			continue
		}

		check := tinymon.Check{
			HostAddress:     addr,
			Type:            "status",
			IntervalSeconds: c.CheckInterval,
			Enabled:         1,
		}
		if err := a.tm.UpsertCheck(check); err != nil {
			log.Printf("ERROR upsert check %s/status: %v", addr, err)
		}

		a.syncHTTPCheck(addr, c)
		a.syncCertCheck(addr, c)

		status, msg := containerStatus(c)
		results = append(results, tinymon.Result{
			HostAddress: addr,
			CheckType:   "status",
			Status:      status,
			Message:     msg,
		})
	}

	if len(results) > 0 {
		if err := a.tm.PushBulk(results); err != nil {
			log.Printf("ERROR push bulk: %v", err)
		}
	}

	// Cleanup: remove hosts that no longer have containers
	for addr := range a.known {
		if !seen[addr] {
			log.Printf("Removing stale host %s", addr)
			if err := a.tm.DeleteHost(addr); err != nil {
				log.Printf("ERROR delete host %s: %v", addr, err)
			}
		}
	}
	a.known = seen

	log.Printf("Synced %d containers, %d results pushed", len(containers), len(results))
}

func (a *Agent) syncHTTPCheck(addr string, c docker.ContainerInfo) {
	if c.HTTPConfig == nil {
		return
	}

	cfg := make(map[string]interface{})
	if c.HTTPConfig.URL != "" {
		cfg["url"] = c.HTTPConfig.URL
	} else {
		cfg["url"] = c.HTTPConfig.Path
		cfg["port"] = c.HTTPConfig.Port
	}
	if c.HTTPConfig.ExpectedStatus > 0 && c.HTTPConfig.ExpectedStatus != 200 {
		cfg["expected_status"] = c.HTTPConfig.ExpectedStatus
	}

	check := tinymon.Check{
		HostAddress:     addr,
		Type:            "http",
		Config:          cfg,
		IntervalSeconds: c.CheckInterval,
		Enabled:         1,
	}
	if err := a.tm.UpsertCheck(check); err != nil {
		log.Printf("ERROR upsert http check %s: %v", addr, err)
	}
}

func (a *Agent) syncCertCheck(addr string, c docker.ContainerInfo) {
	if c.CertConfig == nil {
		return
	}

	cfg := map[string]interface{}{
		"host": c.CertConfig.Host,
		"port": c.CertConfig.Port,
	}

	check := tinymon.Check{
		HostAddress:     addr,
		Type:            "certificate",
		Config:          cfg,
		IntervalSeconds: 3600,
		Enabled:         1,
	}
	if err := a.tm.UpsertCheck(check); err != nil {
		log.Printf("ERROR upsert certificate check %s: %v", addr, err)
	}
}

func (a *Agent) address(containerName string) string {
	return "docker://" + a.agentName + "/" + strings.TrimPrefix(containerName, "/")
}

func containerStatus(c docker.ContainerInfo) (string, string) {
	switch c.State {
	case "running":
		switch c.Health {
		case "unhealthy":
			return "critical", "Container running (unhealthy)"
		case "starting":
			return "warning", "Container running (health: starting)"
		case "healthy":
			return "ok", "Container running (healthy)"
		default:
			return "ok", "Container running"
		}
	case "restarting":
		return "warning", "Container restarting"
	case "exited":
		return "critical", fmt.Sprintf("Container exited (code %d)", c.ExitCode)
	default:
		return "critical", fmt.Sprintf("Container %s", c.State)
	}
}
