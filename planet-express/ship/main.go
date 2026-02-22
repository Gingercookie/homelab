// ship-service/main.go
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Ship struct {
	Name      string     `json:"name"`
	Available bool       `json:"available"`
	Lock      sync.Mutex `json:"-"`
}

type ShipInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

var (
	fleet = []Ship{
		{"Old Bessie", true, sync.Mutex{}},
		{"The Dinghy", true, sync.Mutex{}},
		{"Leela's Cruiser", true, sync.Mutex{}},
	}

	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_ship_requests_received_total",
			Help: "The total number of requests received by the ship service",
		},
		[]string{"method"},
	)

	requestsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_ship_requests_processed_total",
			Help: "The total number of requests handled (processed) by the ship service",
		},
		[]string{"method", "code"},
	)
)

func getStatus(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request for ship status")
	if r.Method != http.MethodGet {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ship := r.URL.Query().Get("ship")
	if ship == "" {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusBadRequest)).Inc()
		http.Error(w, "Missing ship name in status request", http.StatusBadRequest)
		return
	}

	found := false
	for i := range fleet {
		if fleet[i].Lock.TryLock() {
			if fleet[i].Name == ship {
				found = true
				json.NewEncoder(w).Encode(ShipInfo{fleet[i].Name, fleet[i].Available})
			}

			fleet[i].Lock.Unlock()
			if found {
				requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
				return
			}
		}
	}

	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusNotFound)).Inc()
	http.NotFound(w, r)
}

func reserveShip(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to reserve ship")

	found := false
	for i := range fleet {
		if fleet[i].Lock.TryLock() {
			if fleet[i].Available {
				found = true
				slog.Info("Ship is available", "name", fleet[i].Name)
				fleet[i].Available = false
				slog.Info("Ship has been reserved", "name", fleet[i].Name)
				json.NewEncoder(w).Encode(ShipInfo{fleet[i].Name, fleet[i].Available})
			}

			fleet[i].Lock.Unlock()
			if found {
				requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
				return
			}
		}
	}

	slog.Warn("No ship is available")
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
	http.Error(w, "No ship available", http.StatusServiceUnavailable)
}

func returnShip(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to return ship")

	var ship ShipInfo
	if err := json.NewDecoder(r.Body).Decode(&ship); err != nil {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		http.Error(w, "Failed to unmarshal data into ship member", http.StatusServiceUnavailable)
		return
	}

	for i := range fleet {
		if fleet[i].Name == ship.Name {
			slog.Info("Returning ship to base", "name", fleet[i].Name)
			fleet[i].Lock.Lock()
			fleet[i].Available = true
			fleet[i].Lock.Unlock()
			slog.Info("Ship returned and is now available", "name", fleet[i].Name)
			requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusNotFound)).Inc()
	http.NotFound(w, r)
}

func main() {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "INFO"
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	prometheus.MustRegister(requestsReceived)
	prometheus.MustRegister(requestsProcessed)

	shipMux := http.NewServeMux()
	shipMux.HandleFunc("/ship/status", getStatus)
	shipMux.HandleFunc("/ship/reserve", reserveShip)
	shipMux.HandleFunc("/ship/return", returnShip)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		slog.Info("Prometheus metrics endpoint running", "addr", ":2112")
		if err := http.ListenAndServe(":2112", metricsMux); err != nil {
			slog.Error("metrics server error", "err", err)
			os.Exit(1)
		}
	}()

	slog.Info("ShipService running", "addr", ":8080")
	if err := http.ListenAndServe(":8080", shipMux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
