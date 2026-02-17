package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/unclesamwk/tinymon-docker-agent/internal/agent"
	"github.com/unclesamwk/tinymon-docker-agent/internal/tinymon"
)

func main() {
	url := os.Getenv("TINYMON_URL")
	apiKey := os.Getenv("TINYMON_API_KEY")
	if url == "" || apiKey == "" {
		log.Fatal("TINYMON_URL and TINYMON_API_KEY are required")
	}

	agentName := os.Getenv("AGENT_NAME")
	if agentName == "" {
		h, err := os.Hostname()
		if err != nil {
			log.Fatal("cannot determine hostname and AGENT_NAME not set")
		}
		agentName = h
	}

	interval := 60
	if v := os.Getenv("INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i >= 10 {
			interval = i
		}
	}

	log.Printf("tinymon-docker-agent starting (agent=%s, interval=%ds, url=%s)", agentName, interval, url)

	tm := tinymon.NewClient(url, apiKey)
	a := agent.New(tm, agentName)

	ctx := context.Background()
	a.Sync(ctx)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.Sync(ctx)
	}
}
