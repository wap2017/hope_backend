package api

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CurlLoggingMiddleware logs curl commands for incoming requests and response info in one line
func CurlLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip static file requests
		if isStaticFileRequest(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Generate unique request ID
		requestID := generateRequestID()

		// Read the request body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("[%s] Error reading request body: %v", requestID, err)
			c.Next()
			return
		}

		// Restore the request body for the next handler
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Generate the curl command
		curlCommand := generateCurlCommand(c.Request, bodyBytes)

		// Create a response writer to capture response data
		writer := &responseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = writer

		// Record start time
		startTime := time.Now()

		// Process the request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log everything in one line
		log.Printf("[%s] %s | Status: %d | Duration: %v | Response: %s",
			requestID,
			curlCommand,
			writer.status,
			duration,
			writer.body.String())
	}
}

// isStaticFileRequest checks if the request is for static files
func isStaticFileRequest(path string) bool {
	staticPaths := []string{
		"/hope/file/",
		"/uploads/",
		"/static/",
		"/assets/",
		"/css/",
		"/js/",
		"/images/",
		"/favicon.ico",
	}

	for _, staticPath := range staticPaths {
		if strings.HasPrefix(path, staticPath) {
			return true
		}
	}

	// Check for common static file extensions
	staticExtensions := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2", ".ttf", ".eot"}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	return false
}

// generateRequestID creates a unique identifier for each request
func generateRequestID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// responseWriter wraps gin.ResponseWriter to capture response data
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// generateCurlCommand creates a curl command string from the HTTP request
func generateCurlCommand(req *http.Request, body []byte) string {
	var curlCmd strings.Builder

	// Start with curl command
	curlCmd.WriteString("curl -X ")
	curlCmd.WriteString(req.Method)

	// Add URL
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	curlCmd.WriteString(fmt.Sprintf(" '%s://%s%s'", scheme, req.Host, req.URL.String()))

	// Add headers (no masking)
	for name, values := range req.Header {
		for _, value := range values {
			curlCmd.WriteString(fmt.Sprintf(" -H '%s: %s'", name, value))
		}
	}

	// Add request body if present
	if len(body) > 0 {
		// Try to format JSON body nicely, but escape single quotes
		bodyStr := string(body)
		bodyStr = strings.ReplaceAll(bodyStr, "'", "'\"'\"'")
		curlCmd.WriteString(fmt.Sprintf(" -d '%s'", bodyStr))
	}

	return curlCmd.String()
}
