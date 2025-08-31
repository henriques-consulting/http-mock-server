package handler

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

// LoggingMiddleware returns middleware that logs all HTTP requests and responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Read and log request body
			var requestBody []byte
			if r.Body != nil {
				requestBody, _ = io.ReadAll(r.Body)
				r.Body.Close()
				// Restore the body for the next handler
				r.Body = io.NopCloser(bytes.NewReader(requestBody))
			}

			// Wrap the response writer to capture response data
			lw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}

			// Call the next handler
			next.ServeHTTP(lw, r)

			// Format request headers
			var reqHeadersBuf bytes.Buffer
			for name, values := range r.Header {
				for _, value := range values {
					reqHeadersBuf.WriteString("\n        " + name + ": " + value)
				}
			}

			// Format request body
			reqBodyStr := string(requestBody)
			if reqBodyStr == "" {
				reqBodyStr = "(empty)"
			}

			// Format response headers
			var respHeadersBuf bytes.Buffer
			for name, values := range lw.Header() {
				for _, value := range values {
					respHeadersBuf.WriteString("\n        " + name + ": " + value)
				}
			}
			// Format response body
			respBodyStr := lw.body.String()
			if respBodyStr == "" {
				respBodyStr = "(empty)"
			}

			// Log the complete request/response with improved readability
			log.Printf(
				`
==================== HTTP REQUEST ====================
Remote IP: %s
Request:
    Method: %s
    URI: %s
    Headers: %s
    Body: %s
Response:
    Status: %d
    Headers: %s
    Body: %s
=====================================================
`,
				r.RemoteAddr,
				r.Method,
				r.RequestURI,
				reqHeadersBuf.String(),
				reqBodyStr,
				lw.statusCode,
				respHeadersBuf.String(),
				respBodyStr,
			)
		},
	)
}

// loggingResponseWriter wraps http.ResponseWriter to capture response data
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *loggingResponseWriter) Write(data []byte) (int, error) {
	// Write to both the response body buffer and the actual response
	lw.body.Write(data)
	return lw.ResponseWriter.Write(data)
}
