package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"sort"
)

// logBodyLimit is the maximum number of bytes logged for request and response bodies.
// Bodies larger than this are replaced with an omission message in the log.
const logBodyLimit = 1024 * 1024

var bodyOmittedNotice = fmt.Sprintf("(omitted, body exceeds %d bytes)", logBodyLimit)

// LoggingMiddleware returns middleware that logs all HTTP requests and responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Read up to logBodyLimit+1 bytes for log formatting.
			// The original body is restored (including unread bytes) for the next handler.
			var requestBody []byte
			if r.Body != nil {
				requestBody, _ = io.ReadAll(io.LimitReader(r.Body, logBodyLimit+1))
				r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(requestBody), r.Body))
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
			var reqHeaderNames []string
			for name := range r.Header {
				reqHeaderNames = append(reqHeaderNames, name)
			}
			sort.Strings(reqHeaderNames)
			for _, name := range reqHeaderNames {
				for _, value := range r.Header[name] {
					reqHeadersBuf.WriteString("\n        " + name + ": " + value)
				}
			}

			reqBodyStr := formatLogBody(requestBody, r.Header.Get("Content-Type"))

			// Format response headers
			var respHeadersBuf bytes.Buffer
			var respHeaderNames []string
			for name := range lw.Header() {
				respHeaderNames = append(respHeaderNames, name)
			}
			sort.Strings(respHeaderNames)
			for _, name := range respHeaderNames {
				for _, value := range lw.Header()[name] {
					respHeadersBuf.WriteString("\n        " + name + ": " + value)
				}
			}
			respBodyStr := formatLogBody(lw.body.Bytes(), lw.Header().Get("Content-Type"))

			// Log the complete request/response with improved readability
			log.Printf(
				`
==================== HTTP REQUEST ====================
Remote IP: %s
Request:
    Method: %s
    URI: %s
    Headers: %s
    Body:%s
Response:
    Status: %d
    Headers: %s
    Body:%s
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

// formatLogBody prepends its own separator (a space, or a newline before an
// indented block for JSON) so the caller's "Body:" label needs no trailing space.
func formatLogBody(body []byte, contentType string) string {
	if len(body) > logBodyLimit {
		return " " + bodyOmittedNotice
	}
	if len(body) == 0 {
		return " (empty)"
	}
	if mediaType, _, err := mime.ParseMediaType(contentType); err == nil && mediaType == "application/json" {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "        ", "  "); err == nil {
			return "\n        " + pretty.String()
		}
	}
	return " " + string(body)
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
	// Buffer up to logBodyLimit+1 bytes for logging; the full body is always sent to the client.
	if lw.body.Len() <= logBodyLimit {
		remaining := logBodyLimit + 1 - lw.body.Len()
		if len(data) <= remaining {
			lw.body.Write(data)
		} else {
			lw.body.Write(data[:remaining])
		}
	}
	return lw.ResponseWriter.Write(data)
}
