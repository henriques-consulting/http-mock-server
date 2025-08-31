# HTTP Mock Server

A simple and configurable HTTP mock server written in Go that allows you to define mock responses based on request path, method, and headers.

## Features

- **Flexible Request Matching**: Match requests by path, HTTP method, and headers using regex patterns
- **Header Regex Matching**: Use regular expressions to match header values (e.g., `.*` for any value, `application/.*` for application types)
- **Multiple Header Support**: Match against multiple headers simultaneously - all headers must match for the rule to apply
- **Configurable Responses**: Define custom response bodies, status codes, and headers
- **Request/Response Logging**: Comprehensive logging of all HTTP interactions
- **Graceful Shutdown**: Proper cleanup on termination signals
- **Health Check Endpoint**: Built-in `/health` endpoint for monitoring

## Quick Start

1. **Create a configuration file** (`config/config.yaml`):

```yaml
server:
  port: "8080"

requests:
  # Match exact JSON content-type
  - path: /api/users
    headers:
      Content-Type: "application/json"
    method: GET
    response:
      status-code: 200
      headers:
        Content-Type: "application/json"
      body:
        users:
          - id: 1
            name: "John Doe"
          - id: 2
            name: "Jane Smith"

  # Match any application type using regex
  - path: /api/data
    headers:
      Content-Type: "application/.*"
      Authorization: "Bearer .*"
    method: POST
    response:
      status-code: 201
      headers:
        Content-Type: "application/json"
      body:
        message: "Data created successfully"

  # Match any content-type (wildcard)
  - path: /health
    headers:
      Content-Type: ".*"
    method: GET
    response:
      status-code: 200
      body: "OK"

  # No header matching (matches any request to this path/method)
  - path: /ping
    method: GET
    response:
      status-code: 200
      body: "pong"
```

2. **Build and run the server**:

```bash
# Build the application
go build -o http-mock-server ./cmd

# Run the server
./http-mock-server
```

3. **Test your mock endpoints**:

```bash
# Test the users endpoint
curl -H "Content-Type: application/json" http://localhost:8080/api/users

# Test the data endpoint (requires Authorization header)
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/data

# Test the health endpoint
curl http://localhost:8080/health

# Test the ping endpoint
curl http://localhost:8080/ping
```

## Configuration Reference

### Request Rules

Each request rule supports the following fields:

- `path` (required): The exact path to match
- `method` (optional): HTTP method (defaults to GET)
- `headers` (optional): Map of header name to regex pattern. All headers must match for the rule to apply
- `body` (optional): Regex pattern to match against request body
- `response` (required): Response specification

### Response Specification

- `status-code` (optional): HTTP status code (defaults to 200)
- `headers` (optional): Map of response headers to set
- `body` (optional): Response body (can be string or structured data for JSON)
- `content-type` (optional): Response content type (defaults to application/json)

### Header Matching Examples

```yaml
headers:
  # Exact match
  Content-Type: "application/json"

  # Any value (wildcard)
  Accept: ".*"

  # Specific patterns
  User-Agent: "Mozilla.*"

  # Multiple alternatives
  Accept: "(application/json)|(application/xml)"

  # Case-sensitive patterns
  Authorization: "Bearer [A-Za-z0-9]+"
```

## License

This project is licensed under the MIT License. Copyright © 2025 Henriques Consulting AB.

## Contributing
This is a read-only repository. We do not accept external contributions.
