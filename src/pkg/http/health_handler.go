package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HealthStatus represents the health check response
type HealthStatus struct {
	Status             string    `json:"status"`               // "healthy", "degraded", "unhealthy"
	Timestamp          time.Time `json:"timestamp"`            // Current timestamp
	Uptime             string    `json:"uptime"`               // Application uptime
	GatewayOnline      bool      `json:"gateway_online"`       // Gateway connection status
	LastSuccessfulPoll string    `json:"last_successful_poll"` // Time since last successful poll
	ErrorCount         int       `json:"error_count"`          // Current error count in window
	SuccessCount       int       `json:"success_count"`        // Current success count in window
	Version            string    `json:"version,omitempty"`    // Application version (optional)
}

// HealthChecker interface for providing health information
type HealthChecker interface {
	IsOnline() bool
	GetLastSuccessTime() time.Time
	GetErrorCount() int
	GetSuccessCount() int
}

// HealthHandler provides HTTP health check endpoint
type HealthHandler struct {
	startTime     time.Time
	healthChecker HealthChecker
	version       string
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(healthChecker HealthChecker, version string) *HealthHandler {
	return &HealthHandler{
		startTime:     time.Now(),
		healthChecker: healthChecker,
		version:       version,
	}
}

// ServeHTTP implements http.Handler interface for /health endpoint
func (hh *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := hh.getHealthStatus()

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Determine HTTP status code based on health
	statusCode := http.StatusOK
	if status.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if status.Status == "degraded" {
		statusCode = http.StatusOK // Still OK, but with warning
	}

	w.WriteHeader(statusCode)

	// Encode and send JSON response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(status); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode health status: %v", err), http.StatusInternalServerError)
	}
}

// getHealthStatus determines current health status
func (hh *HealthHandler) getHealthStatus() HealthStatus {
	now := time.Now()
	uptime := now.Sub(hh.startTime)

	isOnline := hh.healthChecker.IsOnline()
	lastSuccess := hh.healthChecker.GetLastSuccessTime()
	errorCount := hh.healthChecker.GetErrorCount()
	successCount := hh.healthChecker.GetSuccessCount()

	// Calculate time since last successful poll
	var lastPollStr string
	if !lastSuccess.IsZero() {
		timeSince := now.Sub(lastSuccess)
		if timeSince < time.Minute {
			lastPollStr = fmt.Sprintf("%d seconds ago", int(timeSince.Seconds()))
		} else if timeSince < time.Hour {
			lastPollStr = fmt.Sprintf("%d minutes ago", int(timeSince.Minutes()))
		} else {
			lastPollStr = fmt.Sprintf("%d hours ago", int(timeSince.Hours()))
		}
	} else {
		lastPollStr = "never"
	}

	// Determine overall status
	status := "healthy"
	if !isOnline {
		status = "unhealthy"
	} else if errorCount > 0 {
		total := errorCount + successCount
		if total > 0 {
			errorRate := float64(errorCount) / float64(total) * 100.0
			if errorRate > 50.0 {
				status = "unhealthy"
			} else if errorRate > 20.0 {
				status = "degraded"
			}
		}
	}

	return HealthStatus{
		Status:             status,
		Timestamp:          now,
		Uptime:             formatDuration(uptime),
		GatewayOnline:      isOnline,
		LastSuccessfulPoll: lastPollStr,
		ErrorCount:         errorCount,
		SuccessCount:       successCount,
		Version:            hh.version,
	}
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%d days %d hours", days, hours)
	}
}

// StartHealthServer starts an HTTP server for health checks
func StartHealthServer(handler *HealthHandler, port int) error {
	http.Handle("/health", handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html>
<head><title>MQTT-Modbus Bridge</title></head>
<body>
<h1>MQTT-Modbus Bridge</h1>
<ul>
<li><a href="/health">Health Check</a></li>
<li><a href="/metrics">Metrics</a> (if enabled)</li>
</ul>
</body>
</html>`)
	})

	addr := fmt.Sprintf(":%d", port)

	// Create server with secure timeout settings (gosec G114)
	server := &http.Server{
		Addr:              addr,
		ReadTimeout:       15 * time.Second, // Max time to read request
		ReadHeaderTimeout: 10 * time.Second, // Max time to read headers
		WriteTimeout:      15 * time.Second, // Max time to write response
		IdleTimeout:       60 * time.Second, // Max time for keep-alive connections
	}

	return server.ListenAndServe()
}
