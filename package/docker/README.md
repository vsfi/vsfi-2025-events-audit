# Docker Configuration for Events Audit

This directory contains Docker configurations for containerizing the Events Audit application.

## Docker Files

### 1. Dockerfile.binary
- **Purpose**: Creates a Docker image using a pre-built binary
- **Base Image**: Alpine Linux 3.19
- **Size**: ~33MB
- **Use Case**: Production deployments with pre-compiled binaries

**Features:**
- Uses pre-built binary from `make build`
- Minimal Alpine Linux base image
- Non-root user execution
- CA certificates for HTTPS connections
- Timezone data support

### 2. Dockerfile.multistage
- **Purpose**: Multi-stage build that compiles the application inside Docker
- **Base Images**: golang:1.23-alpine (builder) + scratch (runtime)
- **Size**: ~9MB
- **Use Case**: CI/CD pipelines, development environments

**Features:**
- Multi-stage build for minimal image size
- Static compilation with no external dependencies
- Scratch base image for maximum security
- Built-in dependency management

## Building Images

### Using Makefile Commands

```bash
# Build binary-based image (requires pre-built binary)
make docker-build-binary

# Build multistage image (builds from source)
make docker-build-multistage

# Build both images
make docker-build-all
```

### Manual Build Commands

```bash
# Binary-based build
make build  # First build the binary
cp build/events-audit package/docker/
docker build -f package/docker/Dockerfile.binary -t events-audit:binary package/docker/
rm package/docker/events-audit

# Multistage build
docker build -f package/docker/Dockerfile.multistage -t events-audit:multistage .
```

## Running Containers

### Direct Docker Run

```bash
# Run binary image
docker run --rm -p 8080:8080 events-audit:binary \
  --audit nats \
  --audit-nats-addr nats://localhost:4222

# Run multistage image
docker run --rm -p 8080:8080 events-audit:multistage \
  --audit nats \
  --audit-nats-addr nats://localhost:4222
```

### Using Docker Compose

```bash
# Start services (builds binary image automatically)
make compose-up

# Stop services
make compose-down

# View logs
make compose-logs

# Restart services
make compose-restart
```

## Environment Variables

The application supports configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `AUDIT_LISTNER_AUDIT` | `nope` | Audit type (nats/nsq/nope) |
| `AUDIT_LISTNER_AUDIT_NATS_ADDR` | - | NATS server address |
| `AUDIT_LISTNER_LOG_LEVEL` | `debug` | Log level |
| `AUDIT_LISTNER_LOG_FORMAT` | `text` | Log format (text/json) |
| `AUDIT_LISTNER_AUDIT_TOPIC` | `accountats` | Audit topic |
| `AUDIT_LISTNER_AUDIT_STREAM_NAME` | `EVENTS` | JetStream stream name |
| `AUDIT_LISTNER_AUDIT_CONSUMER_NAME` | `events-audit-consumer` | Consumer name |
| `AUDIT_LISTNER_AUDIT_DURABLE_NAME` | `events-audit-durable` | Durable consumer name |

## Docker Compose Configuration

The `docker-compose.yml` includes:

- **events-audit**: Application container using binary image
- **nats**: NATS JetStream server for message processing
- **Networking**: Internal network for service communication
- **Health checks**: Ensures NATS is ready before starting the app
- **Persistent storage**: NATS data persistence across restarts

## Security Features

Both Docker images implement security best practices:

- **Non-root user**: Applications run as user `appuser` (UID 1001)
- **Minimal attack surface**: Alpine/scratch base images
- **No package managers**: Reduced security vulnerabilities
- **Read-only filesystems**: Containers don't write to the filesystem
- **Least privilege**: Only necessary permissions granted

## Troubleshooting

### Common Issues

1. **Container fails to start**
   ```bash
   # Check logs
   docker-compose logs events-audit
   
   # Verify NATS connectivity
   docker-compose logs nats
   ```

2. **Permission denied errors**
   ```bash
   # Ensure binary is executable
   chmod +x build/events-audit
   ```

3. **Connection refused to NATS**
   ```bash
   # Verify NATS is healthy
   docker-compose ps
   
   # Check NATS logs
   docker-compose logs nats
   ```

### Debugging

```bash
# Run container interactively
docker run --rm -it --entrypoint /bin/sh events-audit:binary

# For multistage (scratch-based), use debugging image
docker run --rm -it alpine:3.19 /bin/sh
```

## Image Comparison

| Feature | Binary Image | Multistage Image |
|---------|--------------|------------------|
| Size | ~33MB | ~9MB |
| Build Time | Fast | Slower |
| Dependencies | Pre-built binary | None |
| Security | Good | Excellent |
| Debugging | Easy | Difficult |
| Use Case | Production | CI/CD |

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make docker-build-binary` | Build binary-based image |
| `make docker-build-multistage` | Build multistage image |
| `make docker-build-all` | Build both images |
| `make docker-run-binary` | Run binary image |
| `make docker-run-multistage` | Run multistage image |
| `make docker-clean` | Remove Docker images |
| `make compose-up` | Start docker-compose services |
| `make compose-down` | Stop docker-compose services |
| `make compose-logs` | View service logs |
| `make compose-restart` | Restart services |
| `make compose-test` | Test running services |