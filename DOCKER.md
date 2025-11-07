# Docker Deployment Guide

This guide explains how to run `mcp-language-server` in a Docker container with HTTP/SSE transport.

## Overview

The Docker image includes:
- `mcp-language-server` binary
- `clangd` language server (v19.1.5)
- All necessary runtime dependencies

## Benefits of Docker Deployment

### HTTP/SSE Transport
- **Better networking**: Works naturally with container orchestration
- **Load balancing**: Can use standard HTTP load balancers
- **Monitoring**: Standard HTTP metrics and health checks
- **Scalability**: Easier to scale horizontally
- **Firewall-friendly**: Uses standard HTTP ports

### Performance Optimizations
- **Resource limits**: Control memory and CPU usage
- **clangd caching**: Persistent cache volumes for faster startup
- **Pre-installed binary**: No need to install clangd separately

## Quick Start

### Using docker-compose (Recommended)

1. Update `docker-compose.yml` to mount your workspace:
```yaml
volumes:
  - /path/to/your/project:/workspace:ro
```

2. Start the service:
```bash
docker-compose up -d
```

3. Access the server at `http://localhost:8080`

### Using Docker directly

Build the image:
```bash
docker build -t mcp-language-server .
```

Run the container:
```bash
docker run -d \
  --name mcp-language-server \
  -p 8080:8080 \
  -v /path/to/your/project:/workspace:ro \
  -v clangd-cache:/root/.cache/clangd \
  -e LOG_LEVEL=INFO \
  --memory=2g \
  --cpus=2 \
  mcp-language-server
```

## Configuration

### Environment Variables

- `TRANSPORT`: Transport type (`http` or `stdio`). Default: `http`
- `PORT`: HTTP port for SSE transport. Default: `8080`
- `LOG_LEVEL`: Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`). Default: `INFO`

### Volume Mounts

- `/workspace`: Mount your project directory here (read-only recommended)
- `/root/.cache/clangd`: Optional cache for better performance

### Resource Limits

Recommended limits based on project size:

**Small projects** (< 1000 files):
- Memory: 512MB - 1GB
- CPU: 0.5 - 1 core

**Medium projects** (1000 - 10,000 files):
- Memory: 1GB - 2GB
- CPU: 1 - 2 cores

**Large projects** (> 10,000 files):
- Memory: 2GB - 4GB
- CPU: 2 - 4 cores

Example with custom limits:
```bash
docker run -d \
  --name mcp-language-server \
  --memory=4g \
  --cpus=4 \
  -p 8080:8080 \
  -v /path/to/large/project:/workspace:ro \
  mcp-language-server
```

## Health Checks

The container includes a health check endpoint at `/health`:

```bash
curl http://localhost:8080/health
```

Response:
```json
{"status":"ok","version":"v0.0.2"}
```

## Advanced Configuration

### Custom clangd Arguments

To pass custom arguments to clangd, you'll need to override the entrypoint:

```bash
docker run -d \
  --name mcp-language-server \
  -p 8080:8080 \
  -v /path/to/project:/workspace:ro \
  --entrypoint /bin/sh \
  mcp-language-server \
  -c "/app/mcp-language-server --workspace /workspace --lsp clangd --transport http --port 8080 -- --background-index --limit-results=100"
```

### Using with compile_commands.json

Mount your build directory containing `compile_commands.json`:

```bash
docker run -d \
  --name mcp-language-server \
  -p 8080:8080 \
  -v /path/to/project:/workspace:ro \
  -v /path/to/project/build:/workspace/build:ro \
  --entrypoint /bin/sh \
  mcp-language-server \
  -c "/app/mcp-language-server --workspace /workspace --lsp clangd --transport http --port 8080 -- --compile-commands-dir=/workspace/build"
```

## Performance Optimization

### Pre-built Index

For faster startup on repeated runs, use a persistent volume for clangd cache:

```yaml
volumes:
  clangd-cache:
    driver: local
```

### Memory Management

clangd can be memory-intensive. Use these flags to limit memory usage:

```bash
docker run -d \
  --name mcp-language-server \
  --memory=2g \
  -p 8080:8080 \
  -v /path/to/project:/workspace:ro \
  --entrypoint /bin/sh \
  mcp-language-server \
  -c "/app/mcp-language-server --workspace /workspace --lsp clangd --transport http --port 8080 -- --limit-results=50 --background-index-priority=low"
```

### Recommended clangd Flags

For better performance and lower memory usage:
- `--background-index`: Enable background indexing
- `--background-index-priority=low`: Lower priority for indexing
- `--limit-results=N`: Limit number of results returned
- `--compile-commands-dir=PATH`: Use compile_commands.json for precise indexing

## Connecting MCP Clients

### HTTP/SSE Transport

Configure your MCP client to connect via HTTP:

```json
{
  "mcpServers": {
    "language-server": {
      "url": "http://localhost:8080",
      "transport": "sse"
    }
  }
}
```

### Stdio Transport (Legacy)

If you need stdio transport, set `TRANSPORT=stdio` and use docker exec:

```bash
docker run -d \
  --name mcp-language-server \
  -v /path/to/project:/workspace:ro \
  -e TRANSPORT=stdio \
  mcp-language-server

# Connect via stdio
docker exec -i mcp-language-server /app/mcp-language-server --workspace /workspace --lsp clangd
```

## Troubleshooting

### Container won't start
Check logs:
```bash
docker logs mcp-language-server
```

### High memory usage
- Reduce memory limits
- Add clangd flags: `--limit-results=50 --background-index-priority=low`
- Use `compile_commands.json` for precise indexing

### Slow indexing
- Use persistent cache volume
- Increase CPU allocation
- Pre-build index before running

### Health check failing
```bash
# Check if server is running
docker exec mcp-language-server ps aux

# Test health endpoint
docker exec mcp-language-server wget -O- http://localhost:8080/health
```

## Multi-Language Support

To support multiple language servers in one container, you would need to:
1. Install additional language servers in the Dockerfile
2. Run multiple instances with different LSP commands
3. Use different ports for each instance

This is not currently automated but can be done manually.

## Production Deployment

For production use, consider:
1. Using a reverse proxy (nginx, traefik) for TLS termination
2. Setting up monitoring and alerting
3. Using orchestration (Kubernetes, Docker Swarm)
4. Implementing authentication/authorization
5. Setting up log aggregation

Example with reverse proxy:
```yaml
version: '3.8'
services:
  mcp-language-server:
    build: .
    expose:
      - "8080"
    networks:
      - internal
  
  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/nginx/certs
    networks:
      - internal
    depends_on:
      - mcp-language-server

networks:
  internal:
```

## Security Considerations

- Mount workspace as read-only (`:ro`) when possible
- Use non-root user (TODO: Add to Dockerfile)
- Limit resource usage with `--memory` and `--cpus`
- Use TLS for production deployments
- Implement authentication for HTTP endpoints
- Keep clangd updated for security patches

## Future Enhancements

Potential improvements:
- [ ] clangd-index-server integration for distributed indexing
- [ ] Pre-built index files in the image
- [ ] Multi-language server support in one container
- [ ] Metrics endpoint (Prometheus)
- [ ] WebSocket transport option
- [ ] Authentication/authorization middleware
