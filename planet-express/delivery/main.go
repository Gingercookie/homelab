// delivery-service/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	url := fmt.Sprintf("%s/crew/reserve", crewServiceURL)
	slog.Debug("Sending request to crew service", "url", url)
	resp, err := http.Get(url)
	if err != nil {
		return CrewMember{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	slog.Debug("Got response from crew service")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return CrewMember{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}
	slog.Debug("Parsed body of response from crew service", "body", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return CrewMember{}, resp.StatusCode, fmt.Errorf("crew service error: %s", bodyBytes)
	}
	slog.Debug("Crew service status code OK")

	var crew CrewMember
	if err := json.Unmarshal(bodyBytes, &crew); err != nil {
		return CrewMember{}, resp.StatusCode, err
	}
	slog.Debug("Unmarshaled crew member", "name", crew.Name)

	return crew, http.StatusOK, nil
}

func reserveShip() (ShipInfo, int, error) {
	url := fmt.Sprintf("%s/ship/reserve", shipServiceURL)
	slog.Debug("Sending request to ship service", "url", url)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return ShipInfo{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	slog.Debug("Got response from ship service")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ShipInfo{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}
	slog.Debug("Parsed body of response from ship service", "body", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return ShipInfo{}, resp.StatusCode, fmt.Errorf("ship reservation failed: %s", string(bodyBytes))
	}

	slog.Debug("Ship service status code OK")
	var ship ShipInfo
	if err := json.Unmarshal(bodyBytes, &ship); err != nil {
		return ShipInfo{}, resp.StatusCode, err
	}
	slog.Debug("Unmarshaled ship info", "name", ship.Name)

	return ship, resp.StatusCode, nil
}

func createPackage(pkg Package) (Package, int, error) {
	url := fmt.Sprintf("%s/packages", packageServiceURL)
	slog.Debug("Sending request to package service", "url", url)

	data, err := json.Marshal(pkg)
	if err != nil {
		return Package{}, http.StatusInternalServerError, fmt.Errorf("failed to marshal package: %w", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return Package{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	slog.Debug("Got response from package service")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Package{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}

	if resp.StatusCode != http.StatusCreated {
		return Package{}, resp.StatusCode, fmt.Errorf("package creation failed: %s", string(bodyBytes))
	}

	slog.Debug("Package service status code OK")
	var created Package
	if err := json.Unmarshal(bodyBytes, &created); err != nil {
		return Package{}, resp.StatusCode, err
	}
	slog.Debug("Unmarshaled package", "id", created.ID)

	return created, resp.StatusCode, nil
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		return
	}

	slog.Debug("Got request for new delivery")
	var req DeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusBadRequest)).Inc()
		return
	}

	slog.Info("Dispatching request for available crew")
	crew, statusCode, err := requestAvailableCrew()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(statusCode)).Inc()
		return
	}
	slog.Debug("Got crew member", "name", crew.Name)

	slog.Info("Dispatching request to reserve ship")
	ship, statusCode, err := reserveShip()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(statusCode)).Inc()
		return
	}

	slog.Info("Got both crew member and ship")
	if (crew == CrewMember{}) || (ship == ShipInfo{}) {
		http.Error(w, "Unable to get ship or crew", http.StatusServiceUnavailable)
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		return
	}

	slog.Info("Dispatching request to create new package")
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
	slog.Info("Delivery ticket created", "crew", ticket.Crew.Name, "ship", ticket.Ship.Name, "package_id", ticket.Package.ID)

	// Simulate random delivery time
	go func(pkgID string, crew CrewMember, ship ShipInfo) {
		delay := time.Duration(rand.IntN(5)+1) * time.Second
		slog.Info("Ship in-flight", "delay", delay, "package_id", pkgID)
		time.Sleep(delay)

		// Mark package as delivered
		resp, err := http.Get(fmt.Sprintf("%s/packages/update?id=%s&status=delivered", packageServiceURL, pkgID))
		if err != nil {
			slog.Error("Failed to update package status", "err", err)
		} else {
			resp.Body.Close()
			slog.Info("Package marked as delivered", "package_id", pkgID)
		}

		// Delete package from map to prevent boundless growth
		deleteReq, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/packages/delete?id=%s", packageServiceURL, pkgID), nil)
		if err != nil {
			slog.Error("Failed to create delete request", "err", err)
		} else {
			resp, err = http.DefaultClient.Do(deleteReq)
			if err != nil {
				slog.Error("Failed to delete package from list", "err", err)
			} else {
				resp.Body.Close()
				slog.Info("Package deleted successfully", "package_id", pkgID)
			}
		}

		// Return crew to base
		slog.Info("Returning crew member to base", "name", crew.Name)
		data, err := json.Marshal(crew)
		if err != nil {
			slog.Error("Failed to marshal crew member", "name", crew.Name, "err", err)
		} else {
			resp, err = http.Post(fmt.Sprintf("%s/crew/return", crewServiceURL), "application/json", bytes.NewBuffer(data))
			if err != nil {
				slog.Error("Failed to return crew member to base", "name", crew.Name, "err", err)
			} else {
				resp.Body.Close()
				slog.Info("Crew member returned to base", "name", crew.Name)
			}
		}

		// Return ship to base
		slog.Info("Returning ship to base")
		data, err = json.Marshal(ship)
		if err != nil {
			slog.Error("Failed to marshal ship", "name", ship.Name, "err", err)
		} else {
			resp, err = http.Post(fmt.Sprintf("%s/ship/return", shipServiceURL), "application/json", bytes.NewBuffer(data))
			if err != nil {
				slog.Error("Failed to return ship", "err", err)
			} else {
				resp.Body.Close()
				slog.Info("Ship returned to base")
			}
		}
	}(pkg.ID, crew, ship)

	// Send ticket to requester
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
}

func main() {
	levelStr := getEnv("LOG_LEVEL", "INFO")
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	deliveryMux := http.NewServeMux()
	deliveryMux.HandleFunc("/deliveries", handleDelivery)

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

	slog.Info("DeliveryService running", "addr", ":8080")
	if err := http.ListenAndServe(":8080", deliveryMux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
