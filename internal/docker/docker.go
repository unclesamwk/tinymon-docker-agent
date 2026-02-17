package docker

import (
	"context"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type ContainerInfo struct {
	Name           string
	State          string
	Health         string
	ExitCode       int
	Labels         map[string]string
	DisplayName    string
	Topic          string
	CheckInterval  int
	HTTPConfig     *HTTPConfig
	CertConfig     *CertConfig
}

type HTTPConfig struct {
	URL            string
	Path           string
	Port           int
	ExpectedStatus int
}

type CertConfig struct {
	Host string
	Port int
}

func ListEnabled(ctx context.Context) ([]ContainerInfo, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	f := filters.NewArgs()
	f.Add("label", "tinymon.enable=true")

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, err
	}

	var result []ContainerInfo
	for _, c := range containers {
		info := parseContainer(c)
		result = append(result, info)
	}
	return result, nil
}

func parseContainer(c types.Container) ContainerInfo {
	name := strings.TrimPrefix(c.Names[0], "/")
	labels := c.Labels

	info := ContainerInfo{
		Name:          name,
		State:         c.State,
		Labels:        labels,
		DisplayName:   labelStr(labels, "tinymon.name", name),
		Topic:         labelStr(labels, "tinymon.topic", ""),
		CheckInterval: labelInt(labels, "tinymon.check-interval", 60, 30),
	}

	if c.Status != "" && strings.Contains(c.Status, "(health:") {
		if strings.Contains(c.Status, "(healthy)") {
			info.Health = "healthy"
		} else if strings.Contains(c.Status, "(unhealthy)") {
			info.Health = "unhealthy"
		} else if strings.Contains(c.Status, "(health: starting)") {
			info.Health = "starting"
		}
	}

	if c.State == "exited" {
		// Parse exit code from status like "Exited (1) 2 hours ago"
		if idx := strings.Index(c.Status, "("); idx >= 0 {
			if end := strings.Index(c.Status[idx:], ")"); end >= 0 {
				if code, err := strconv.Atoi(c.Status[idx+1 : idx+end]); err == nil {
					info.ExitCode = code
				}
			}
		}
	}

	if url := labelStr(labels, "tinymon.http.url", ""); url != "" {
		info.HTTPConfig = &HTTPConfig{
			URL:            url,
			ExpectedStatus: labelInt(labels, "tinymon.http.expected-status", 200, 100),
		}
	} else if labelStr(labels, "tinymon.http.path", "") != "" || labelStr(labels, "tinymon.http.port", "") != "" {
		info.HTTPConfig = &HTTPConfig{
			Path:           labelStr(labels, "tinymon.http.path", "/"),
			Port:           labelInt(labels, "tinymon.http.port", 443, 1),
			ExpectedStatus: labelInt(labels, "tinymon.http.expected-status", 200, 100),
		}
	}

	if host := labelStr(labels, "tinymon.certificate.host", ""); host != "" {
		info.CertConfig = &CertConfig{
			Host: host,
			Port: labelInt(labels, "tinymon.certificate.port", 443, 1),
		}
	}

	return info
}

func labelStr(labels map[string]string, key, fallback string) string {
	if v, ok := labels[key]; ok && v != "" {
		return v
	}
	return fallback
}

func labelInt(labels map[string]string, key string, fallback, min int) int {
	if v, ok := labels[key]; ok {
		if i, err := strconv.Atoi(v); err == nil && i >= min {
			return i
		}
	}
	return fallback
}
