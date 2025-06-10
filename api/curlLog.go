package api

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CurlLoggingMiddleware logs curl commands for incoming requests
func CurlLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip static file requests
		if isStaticFileRequest(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Read the request body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			c.Next()
			return
		}

		// Restore the request body for the next handler
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Generate and log the curl command
		curlCommand := generateCurlCommand(c.Request, bodyBytes)
		fmt.Printf("CURL: %s", curlCommand)

		c.Next()
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

	// Add headers (with sensitive data masked)
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
