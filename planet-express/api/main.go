// planetexpress-api/main.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	deliveryServiceURL = getEnv("DELIVERY_SERVICE_URL", "http://delivery-service")
)

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// handleNewDelivery forwards delivery requests to the DeliveryService
func handleNewDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("[INFO] Got request for new delivery")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request", http.StatusBadRequest)
		return
	}

	fmt.Printf("[INFO] Dispatching request to delivery-service at %s\n", deliveryServiceURL)
	resp, err := http.Post(fmt.Sprintf("%s/deliveries", deliveryServiceURL), "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Error contacting DeliveryService: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status":"OK"}`))
}

func main() {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/deliveries", handleNewDelivery)
	apiMux.HandleFunc("/health", healthCheck)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		fmt.Println("Prometheus metrics endpoint running on :2112")
		err := http.ListenAndServe(":2112", metricsMux)
		if err != nil {
			log.Fatalln(err)
		}
	}()

	fmt.Println("PlanetExpressAPI running on :8080")
	http.ListenAndServe(":8080", apiMux)
}
