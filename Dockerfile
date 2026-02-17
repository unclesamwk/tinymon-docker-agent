FROM --platform=$BUILDPLATFORM golang:1.24 AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o tinymon-docker-agent .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/tinymon-docker-agent /tinymon-docker-agent
USER 65532:65532
ENTRYPOINT ["/tinymon-docker-agent"]
