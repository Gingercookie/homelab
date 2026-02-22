// delivery-service/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"math/rand/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type CrewMember struct {
	Name string `json:"name"`
}

type ShipInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

type Package struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Status    string `json:"status"`
	Contents  string `json:"contents"`
}

type DeliveryRequest struct {
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Contents  string `json:"contents"`
}

type DeliveryTicket struct {
	Crew    CrewMember `json:"crew"`
	Ship    ShipInfo   `json:"ship"`
	Package Package    `json:"package"`
}

var (
	crewServiceURL    = getEnv("CREW_SERVICE_URL", "http://crew-service")
	shipServiceURL    = getEnv("SHIP_SERVICE_URL", "http://ship-service")
	packageServiceURL = getEnv("PACKAGE_SERVICE_URL", "http://package-service")

	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_delivery_requests_received_total",
			Help: "The total number of requests received by the delivery service",
		},
		[]string{"method"},
	)

	requestsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_delivery_requests_processed_total",
			Help: "The total number of requests handled (processed) by the delivery service",
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

func requestAvailableCrew() (CrewMember, int, error) {
	fmt.Printf("[DEBUG] Sending request to %s\n", fmt.Sprintf("%s/crew/reserve", crewServiceURL))
	resp, err := http.Get(fmt.Sprintf("%s/crew/reserve", crewServiceURL))
	if err != nil {
		return CrewMember{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Got response from crew service\n")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return CrewMember{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}
	fmt.Printf("[DEBUG] Parsed body of response: %s", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return CrewMember{}, resp.StatusCode, fmt.Errorf("crew service error: %s", bodyBytes)
	}
	fmt.Printf("[DEBUG] Status code is OKAY\n")

	var crew CrewMember
	if err := json.Unmarshal(bodyBytes, &crew); err != nil {
		return CrewMember{}, resp.StatusCode, err
	}
	fmt.Printf("[DEBUG] Unmarshaled into %s\n", crew)

	return crew, http.StatusOK, nil
}

func reserveShip() (ShipInfo, int, error) {
	fmt.Printf("[DEBUG] Sending request to %s\n", fmt.Sprintf("%s/ship/reserve", shipServiceURL))
	resp, err := http.Post(fmt.Sprintf("%s/ship/reserve", shipServiceURL), "application/json", nil)
	if err != nil {
		return ShipInfo{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Got response from ship service\n")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ShipInfo{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}
	fmt.Printf("[DEBUG] Parsed body of response: %s", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return ShipInfo{}, resp.StatusCode, fmt.Errorf("ship reservation failed: %s", string(bodyBytes))
	}

	fmt.Printf("[DEBUG] Status code is OKAY\n")
	var ship ShipInfo
	if err := json.Unmarshal(bodyBytes, &ship); err != nil {
		return ShipInfo{}, resp.StatusCode, err
	}
	fmt.Printf("[DEBUG] Unmarshaled into %v\n", ship)

	return ship, resp.StatusCode, nil
}

func createPackage(pkg Package) (Package, int, error) {
	fmt.Printf("[DEBUG] Sending request to %s\n", fmt.Sprintf("%s/packages", packageServiceURL))

	data, err := json.Marshal(pkg)
	if err != nil {
		return Package{}, http.StatusInternalServerError, fmt.Errorf("failed to marshal package: %w", err)
	}
	resp, err := http.Post(fmt.Sprintf("%s/packages", packageServiceURL), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return Package{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Got response from package service\n")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Package{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}

	if resp.StatusCode != http.StatusCreated {
		return Package{}, resp.StatusCode, fmt.Errorf("package creation failed: %s", string(bodyBytes))
	}

	fmt.Printf("[DEBUG] Status code is OKAY\n")
	var created Package
	if err := json.Unmarshal(bodyBytes, &created); err != nil {
		return Package{}, resp.StatusCode, err
	}
	fmt.Printf("[DEBUG] Unmarshaled into %s\n", created)

	return created, resp.StatusCode, nil
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		return
	}

	fmt.Println("[DEBUG] Got request for new delivery")
	var req DeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusBadRequest)).Inc()
		return
	}

	fmt.Println("[INFO] Dispatching request for available crew")
	crew, statusCode, err := requestAvailableCrew()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(statusCode)).Inc()
		return
	}
	fmt.Printf("[DEBUG] Got crew member %s\n", crew.Name)

	fmt.Println("[INFO] Dispatching request to reserve ship")
	ship, statusCode, err := reserveShip()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(statusCode)).Inc()
		return
	}

	fmt.Println("[INFO] Got both crew member and ship")
	if (crew == CrewMember{}) || (ship == ShipInfo{}) {
		http.Error(w, "Unable to get ship or crew", http.StatusServiceUnavailable)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		return
	}

	fmt.Println("[INFO] Dispatching request to create new package")
	pkg, statusCode, err := createPackage(Package{
		Recipient: req.Recipient,
		Address:   req.Address,
		Contents:  req.Contents,
	})
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(statusCode)).Inc()
		return
	}

	// Build the delivery ticket
	ticket := DeliveryTicket{
		Crew:    crew,
		Ship:    ship,
		Package: pkg,
	}
	fmt.Printf("[INFO] Delivery ticket created: %v\n", ticket)

	// Simulate random delivery time
	go func(pkgID string, crew CrewMember, ship ShipInfo) {
		delay := time.Duration(rand.IntN(5)+1) * time.Second
		fmt.Printf("[INFO] ship in-flight for %v delivering package %s\n", delay, pkgID)
		time.Sleep(delay)

		// Mark package as delivered
		resp, err := http.Get(fmt.Sprintf("%s/packages/update?id=%s&status=delivered", packageServiceURL, pkgID))
		if err != nil {
			fmt.Println("[ERROR] Failed to update package status:", err)
		} else {
			resp.Body.Close()
			fmt.Printf("[INFO] Package %s marked as delivered\n", pkgID)
		}

		// Delete package from map to prevent boundless growth
		deleteReq, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/packages/delete?id=%s", packageServiceURL, pkgID), nil)
		if err != nil {
			fmt.Println("[ERROR] Failed to create delete request:", err)
		} else {
			resp, err = http.DefaultClient.Do(deleteReq)
			if err != nil {
				fmt.Println("[ERROR] Failed to delete package from list:", err)
			} else {
				resp.Body.Close()
				fmt.Printf("[INFO] Package %s deleted successfully\n", pkgID)
			}
		}

		// Return crew to base
		fmt.Printf("[INFO] Returning crew member %s to base", crew.Name)
		data, err := json.Marshal(crew)
		if err != nil {
			fmt.Printf("[ERROR] Failed to marshal crew member %s: %v\n", crew.Name, err)
		} else {
			resp, err = http.Post(fmt.Sprintf("%s/crew/return", crewServiceURL), "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Printf("[ERROR] Failed to return crew member %s to base:%s", crew.Name, err)
			} else {
				resp.Body.Close()
				fmt.Printf("[INFO] Crew member %s returned to base.", crew.Name)
			}
		}

		// Return ship to base
		fmt.Println("[INFO] Returning ship to base")
		data, err = json.Marshal(ship)
		if err != nil {
			fmt.Printf("[ERROR] Failed to marshal ship %s: %v\n", ship.Name, err)
		} else {
			resp, err = http.Post(fmt.Sprintf("%s/ship/return", shipServiceURL), "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Println("[ERROR] Failed to return ship:", err)
			} else {
				resp.Body.Close()
				fmt.Println("[INFO] ship returned to base.")
			}
		}
	}(pkg.ID, crew, ship)

	// Send ticket to requester
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
}

func main() {
	deliveryMux := http.NewServeMux()
	deliveryMux.HandleFunc("/deliveries", handleDelivery)

	prometheus.MustRegister(requestsReceived)
	prometheus.MustRegister(requestsProcessed)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		fmt.Println("[INFO] Prometheus metrics endpoint running on :2112")
		err := http.ListenAndServe(":2112", metricsMux)
		if err != nil {
			log.Fatalln(err)
		}
	}()

	fmt.Println("[INFO] DeliveryService running on :8080")
	if err := http.ListenAndServe(":8080", deliveryMux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
