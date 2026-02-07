# HTTP Mock Server

A simple and configurable HTTP mock server written in Go that allows you to define mock responses based on request path, method, headers, and query parameters.

## Features

- **Flexible Request Matching**: Match requests by path, HTTP method, headers, and query parameters using regex patterns
- **Header Regex Matching**: Use regular expressions to match header values (e.g., `.*` for any value, `application/.*` for application types)
- **Query Parameter Matching**: Match query parameters with exact values or regex patterns
- **Multiple Header Support**: Match against multiple headers simultaneously - all headers must match for the rule to apply
- **Configurable Responses**: Define custom response bodies, status codes, and headers
- **Response Delays**: Simulate slow endpoints with configurable random delays
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

  # Match query parameters
  - path: /search
    method: GET
    queryParams:
      q: ".+"
      page: "[0-9]+"
    response:
      status-code: 200
      headers:
        Content-Type: "application/json"
      body:
        results: []

  # Simulate a slow endpoint with random delay
  - path: /slow-api
    method: GET
    responseDelay:
      min: 500
      max: 1500
    response:
      status-code: 200
      body:
        message: "This response was delayed"
```

2. **Build and run the server**:

```bash
# Build the application
go build -o http-mock-server ./cmd

# Run the server
./http-mock-server
```

or

```bash
# Build a docker image
docker build -t http-mock-server --build-arg=VERSION=$(git rev-parse HEAD) .

# Run the server with
docker run -v "</path/to/folder-with-config.yaml>:/app/config" -p 8080:8080 http-mock-server

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

# Test the search endpoint with query parameters
curl "http://localhost:8080/search?q=test&page=1"

# Test the slow endpoint (will take 500-1500ms to respond)
curl -w "\nTime: %{time_total}s\n" http://localhost:8080/slow-api
```

## Configuration Reference

### Request Rules

Each request rule supports the following fields:

- `path` (required): The exact path to match
- `method` (optional): HTTP method (defaults to GET)
- `headers` (optional): Map of header name to regex pattern. All headers must match for the rule to apply
- `queryParams` (optional): Map of query parameter name to regex pattern. All specified params must match for the rule to apply
- `body` (optional): Regex pattern to match against request body
- `responseDelay` (optional): Delay configuration before sending response (see below)
- `response` (required): Response specification

### Response Specification

- `status-code` (optional): HTTP status code (defaults to 200)
- `headers` (optional): Map of response headers to set
- `body` (optional): Response body (can be string or structured data for JSON)

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

### Query Parameter Matching Examples

```yaml
queryParams:
  # Exact match
  status: "active"

  # Any value (wildcard)
  q: ".*"

  # Numeric values only
  page: "[0-9]+"
  limit: "^(10|25|50|100)$"

  # Optional UUID format
  id: "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
```

Note: If `queryParams` is not specified in a rule, the rule matches requests regardless of their query string. When specified, all listed parameters must be present and match their patterns. Extra query parameters in the request (not listed in the rule) are ignored.

### Response Delay

The `responseDelay` field allows you to simulate slow endpoints by adding a delay before the response is sent. This is useful for testing timeout handling, loading states, and retry logic in your applications.

- `min` (required): Minimum delay in milliseconds
- `max` (required): Maximum delay in milliseconds

The actual delay for each request is randomly chosen between `min` and `max` (inclusive). For a fixed delay, set both values to the same number.

**Constraints:**
- Both `min` and `max` must be non-negative
- `min` must be less than or equal to `max`
- `max` cannot exceed 10,000ms (10 seconds)

```yaml
responseDelay:
  # Random delay between 500ms and 1500ms
  min: 500
  max: 1500

responseDelay:
  # Fixed delay of exactly 1 second
  min: 1000
  max: 1000
```

**Example: Simulating a slow API**

```yaml
requests:
  - path: /api/slow-operation
    method: POST
    responseDelay:
      min: 2000
      max: 5000
    response:
      status-code: 200
      headers:
        Content-Type: "application/json"
      body:
        status: "completed"
        processingTime: "variable"
```

## License

This project is licensed under the MIT License. Copyright © 2025 Henriques Consulting AB.

## Contributing
This is a read-only repository. We do not accept external contributions.
