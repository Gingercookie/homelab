// planetexpress-api/main.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	deliveryServiceURL = getEnv("DELIVERY_SERVICE_URL", "http://delivery-service")

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
	fmt.Println("[INFO] Got request for new delivery")

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

	fmt.Printf("[INFO] Dispatching request to delivery-service at %s\n", deliveryServiceURL)
	resp, err := http.Post(fmt.Sprintf("%s/deliveries", deliveryServiceURL), "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Error contacting DeliveryService: "+err.Error(), http.StatusServiceUnavailable)
		requestsProcessed.WithLabelValues(http.MethodPost, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	requestsProcessed.WithLabelValues(http.MethodPost, strconv.Itoa(resp.StatusCode)).Inc()
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status":"OK"}`))
}

func main() {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/deliveries", handleNewDelivery)
	apiMux.HandleFunc("/health", healthCheck)

	prometheus.MustRegister(requestsReceived)
	prometheus.MustRegister(requestsProcessed)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		err := http.ListenAndServe(":2112", metricsMux)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("[INFO] Prometheus metrics endpoint running on :2112")
	}()

	fmt.Println("PlanetExpressAPI running on :8080")
	http.ListenAndServe(":8080", apiMux)
}
