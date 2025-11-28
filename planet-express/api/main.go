// planetexpress-api/main.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request", http.StatusBadRequest)
		return
	}

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
	http.HandleFunc("/deliveries", handleNewDelivery)
	http.HandleFunc("/health", healthCheck)

	fmt.Println("PlanetExpressAPI running on :8080")
	http.ListenAndServe(":8080", nil)
}
