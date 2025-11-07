# Example MCP Client Configuration for HTTP Transport

This directory contains example configurations for connecting MCP clients to mcp-language-server using HTTP/SSE transport.

## Claude Desktop - HTTP Mode

For Claude Desktop, add this to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "language-server-http": {
      "url": "http://localhost:8080",
      "transport": "sse"
    }
  }
}
```

## Claude Desktop - stdio Mode (Local)

For local development with stdio transport:

```json
{
  "mcpServers": {
    "language-server": {
      "command": "mcp-language-server",
      "args": [
        "--workspace", "/path/to/your/project",
        "--lsp", "clangd",
        "--",
        "--compile-commands-dir=/path/to/build"
      ],
      "env": {
        "PATH": "/usr/local/bin:/usr/bin:/bin",
        "LOG_LEVEL": "INFO"
      }
    }
  }
}
```

## Docker Deployment - Production

For production deployment with docker-compose:

1. Update `docker-compose.yml`:
```yaml
version: '3.8'
services:
  mcp-language-server:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - /path/to/your/project:/workspace:ro
      - clangd-cache:/root/.cache/clangd
    environment:
      - LOG_LEVEL=INFO
      - TRANSPORT=http
      - PORT=8080
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '2.0'
    restart: unless-stopped

volumes:
  clangd-cache:
```

2. Configure your MCP client:
```json
{
  "mcpServers": {
    "language-server-docker": {
      "url": "http://localhost:8080",
      "transport": "sse"
    }
  }
}
```

## Testing the Connection

### Health Check
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"ok","version":"v0.0.2"}
```

### SSE Endpoint
```bash
curl -N http://localhost:8080/sse
```

Should establish an SSE connection.

## Troubleshooting

### Connection Refused
- Check if the server is running: `docker ps` or `ps aux | grep mcp-language-server`
- Verify the port is correct
- Check firewall settings

### Timeout/Slow Response
- Increase resource limits in docker-compose.yml
- Check clangd cache is being used
- Verify compile_commands.json exists

### Authentication Errors
- For production, add nginx reverse proxy with authentication
- See nginx.conf.example for TLS setup
