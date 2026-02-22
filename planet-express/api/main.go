// planetexpress-api/main.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	deliveryServiceURL = getEnv("DELIVERY_SERVICE_URL", "http://planetexpress-delivery")

	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_api_requests_received_total",
			Help: "The total number of requests received by the api",
		},
		[]string{"method"},
	)

	requestsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_api_requests_processed_total",
			Help: "The total number of requests handled (processed) by the api",
		},
		[]string{"method", "code"},
	)
)

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// handleNewDelivery forwards delivery requests to the DeliveryService
func handleNewDelivery(w http.ResponseWriter, r *http.Request) {
	slog.Info("Got request for new delivery")

	requestsReceived.WithLabelValues(r.Method).Inc()

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request", http.StatusBadRequest)
		requestsProcessed.WithLabelValues(http.MethodPost, strconv.Itoa(http.StatusBadRequest)).Inc()
		return
	}

	slog.Info("Dispatching request to delivery-service", "url", deliveryServiceURL)
	resp, err := http.Post(fmt.Sprintf("%s/deliveries", deliveryServiceURL), "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Error contacting DeliveryService: "+err.Error(), http.StatusServiceUnavailable)
		requestsProcessed.WithLabelValues(http.MethodPost, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(w, resp.Body); err != nil {
		slog.Error("Failed to copy response body", "err", err)
	}
	requestsProcessed.WithLabelValues(http.MethodPost, strconv.Itoa(resp.StatusCode)).Inc()
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status":"OK"}`))
}

func main() {
	levelStr := getEnv("LOG_LEVEL", "INFO")
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/deliveries", handleNewDelivery)
	apiMux.HandleFunc("/health", healthCheck)

	prometheus.MustRegister(requestsReceived)
	prometheus.MustRegister(requestsProcessed)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		slog.Info("Prometheus metrics endpoint running", "addr", ":2112")
		err := http.ListenAndServe(":2112", metricsMux)
		if err != nil {
			slog.Error("metrics server error", "err", err)
			os.Exit(1)
		}
	}()

	slog.Info("PlanetExpressAPI running", "addr", ":8080")
	if err := http.ListenAndServe(":8080", apiMux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
