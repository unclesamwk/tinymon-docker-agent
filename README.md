# TinyMon Docker Agent

Docker container monitoring agent for [TinyMon](https://github.com/unclesamwk/TinyMon). Discovers containers via labels and pushes their status to the TinyMon Push API.

Add `tinymon.enable=true` as a label to any Docker container and the agent will automatically create hosts and checks in TinyMon.

## How It Works

The agent polls the Docker socket at a configurable interval, finds all containers with `tinymon.enable=true`, and:

1. **Upserts a host** in TinyMon for each container (`docker://<agent>/<container>`)
2. **Upserts a status check** and pushes the current container state
3. **Optionally creates HTTP and certificate checks** (pull mode -- TinyMon executes these)
4. **Cleans up** hosts for containers that no longer exist

## Container Status Mapping

| Docker State | Health | TinyMon Status | Message |
|-------------|--------|---------------|---------|
| running | healthy | ok | Container running (healthy) |
| running | unhealthy | critical | Container running (unhealthy) |
| running | starting | warning | Container running (health: starting) |
| running | (no healthcheck) | ok | Container running |
| restarting | - | warning | Container restarting |
| exited | - | critical | Container exited (code X) |
| created/paused/dead | - | critical | Container \<state\> |

## Quick Start

### Prerequisites

- A running [TinyMon](https://github.com/unclesamwk/TinyMon) instance
- The **Push API key** (`PUSH_API_KEY` from TinyMon's `.env` file)
- Docker installed on the host you want to monitor

### Step 1: Add the agent to your docker-compose.yml

Add the `tinymon-agent` service to the `docker-compose.yml` on the host you want to monitor.
It needs read-only access to the Docker socket to discover running containers.

```yaml
services:
  tinymon-agent:
    image: unclesamwk/tinymon-docker-agent
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      TINYMON_URL: "https://mon.example.com"   # URL of your TinyMon instance
      TINYMON_API_KEY: "your-push-api-key"     # PUSH_API_KEY from TinyMon .env
      AGENT_NAME: "my-docker-host"             # A unique name for this Docker host
```

### Step 2: Add labels to containers you want to monitor

On any container you want to monitor, add `tinymon.enable: "true"` as a label.
The agent picks up new containers automatically -- no restart of the agent needed.

```yaml
services:
  web:
    image: nginx
    labels:
      tinymon.enable: "true"
      tinymon.name: "My Webserver"
```

### Step 3: Start everything

```bash
docker compose up -d
```

The agent starts polling every 60 seconds (configurable via `INTERVAL`).
Within a minute, your container appears in the TinyMon dashboard.

When a container stops or becomes unhealthy, TinyMon sends an alert.
When it recovers, TinyMon sends a recovery notification.


## Labels

Add these labels to containers you want to monitor:

| Label | Description | Default | Required |
|-------|-------------|---------|----------|
| `tinymon.enable` | Enable monitoring | - | Yes ("true") |
| `tinymon.name` | Display name in TinyMon | Container name | No |
| `tinymon.topic` | Topic/group in TinyMon | Docker/\<agent\> | No |
| `tinymon.check-interval` | Check interval in seconds (min 30) | 60 | No |
| `tinymon.http.url` | Full URL for HTTP check (pull mode) | - | No |
| `tinymon.http.path` | HTTP check path | / | No |
| `tinymon.http.port` | HTTP check port | 443 | No |
| `tinymon.http.expected-status` | Expected HTTP status code | 200 | No |
| `tinymon.certificate.host` | Hostname for certificate check (pull mode) | - | No |
| `tinymon.certificate.port` | Port for certificate check | 443 | No |

**Pull mode**: The agent only creates the check in TinyMon. TinyMon executes the HTTP/certificate check independently.

## Examples

**Basic container monitoring:**

```yaml
services:
  web:
    image: nginx
    labels:
      tinymon.enable: "true"
      tinymon.name: "Webserver"
      tinymon.topic: "production/web"
```

**With HTTP and certificate checks:**

```yaml
services:
  app:
    image: myapp
    labels:
      tinymon.enable: "true"
      tinymon.name: "My Application"
      tinymon.topic: "production/apps"
      tinymon.http.url: "https://app.example.com/health"
      tinymon.http.expected-status: "200"
      tinymon.certificate.host: "app.example.com"
```

**With custom HTTP path and port:**

```yaml
services:
  api:
    image: myapi
    labels:
      tinymon.enable: "true"
      tinymon.name: "API Server"
      tinymon.http.path: "/api/health"
      tinymon.http.port: "8080"
      tinymon.check-interval: "120"
```

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `TINYMON_URL` | TinyMon instance URL | - | Yes |
| `TINYMON_API_KEY` | Push API bearer token | - | Yes |
| `AGENT_NAME` | Agent identifier (used in host addresses) | Hostname | No |
| `INTERVAL` | Poll interval in seconds (min 10) | 60 | No |

## Host Addressing

Each container gets a unique address in TinyMon:

```
docker://<AGENT_NAME>/<container-name>
```

For example, with `AGENT_NAME=prod-01` and a container named `web`:

```
docker://prod-01/web
```

## Development

```bash
# Build
go build ./...

# Run locally
export TINYMON_URL=http://localhost:8001
export TINYMON_API_KEY=your-key
export AGENT_NAME=dev
go run .
```

## Image

Available on Docker Hub as [`unclesamwk/tinymon-docker-agent`](https://hub.docker.com/r/unclesamwk/tinymon-docker-agent) for `linux/amd64` and `linux/arm64`.

## License

MIT
