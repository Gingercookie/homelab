// crew-service/main.go
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

type CrewMember struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Available bool       `json:"available"`
	Lock      sync.Mutex `json:"-"`
}

type CrewResponse struct {
	Name string `json:"name"`
}

var (
	crew = []CrewMember{
		{"Fry", "Delivery Boy", true, sync.Mutex{}},
		{"Leela", "Captain", true, sync.Mutex{}},
		{"Bender", "Bending Unit", true, sync.Mutex{}},
	}

	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_crew_requests_received_total",
			Help: "The total number of requests received by the crew service",
		},
		[]string{"method"},
	)

	requestsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_crew_requests_processed_total",
			Help: "The total number of requests handled (processed) by the crew service",
		},
		[]string{"method", "code"},
	)
)

func reserveCrew(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to reserve a crew member")
	found := false
	for i := range crew {
		if crew[i].Lock.TryLock() {
			if crew[i].Available {
				found = true
				slog.Info("Crew member is available", "name", crew[i].Name)
				crew[i].Available = false
				slog.Info("Crew member has been reserved", "name", crew[i].Name)
				json.NewEncoder(w).Encode(CrewResponse{crew[i].Name})
			}
			// Need to unlock the mutex before we return
			crew[i].Lock.Unlock()
			if found {
				requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
				return
			}
		}
	}

	slog.Warn("No crew is available")
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
	http.Error(w, "No crew available", http.StatusServiceUnavailable)
}

func returnCrew(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to return a crew member")

	var c CrewMember
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusServiceUnavailable)).Inc()
		http.Error(w, "Failed to unmarshal data into crew member", http.StatusServiceUnavailable)
		return
	}

	for i := range crew {
		if crew[i].Name == c.Name {
			crew[i].Lock.Lock()
			crew[i].Available = true
			crew[i].Lock.Unlock()

			slog.Info("Crew member returned successfully", "name", c.Name)
			requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusNotFound)).Inc()
	http.Error(w, "Crew member not found", http.StatusNotFound)
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

	crewMux := http.NewServeMux()
	crewMux.HandleFunc("/crew/reserve", reserveCrew)
	crewMux.HandleFunc("/crew/return", returnCrew)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		slog.Info("Prometheus metrics endpoint running", "addr", ":2112")
		if err := http.ListenAndServe(":2112", metricsMux); err != nil {
			slog.Error("metrics server error", "err", err)
			os.Exit(1)
		}
	}()

	slog.Info("CrewService running", "addr", ":8080")
	if err := http.ListenAndServe(":8080", crewMux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
