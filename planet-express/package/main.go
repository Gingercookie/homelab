// package-service/main.go
package main

import (
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Package struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Status    string `json:"status"` // "pending", "in-transit", "delivered"
	Contents  string `json:"contents"`
}

var (
	packages = make(map[string]Package)
	mu       sync.Mutex

	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_package_requests_received_total",
			Help: "The total number of requests received by the package service",
		},
		[]string{"method"},
	)

	requestsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "planet_express_package_requests_processed_total",
			Help: "The total number of requests handled (processed) by the package service",
		},
		[]string{"method", "code"},
	)
)

func createPackage(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	var pkg Package
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusBadRequest)).Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	pkg.ID = randomID()
	pkg.Status = "pending"
	packages[pkg.ID] = pkg
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusCreated)).Inc()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pkg)
}

func listPackages(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	mu.Lock()
	defer mu.Unlock()
	list := make([]Package, 0, len(packages))
	for _, pkg := range packages {
		list = append(list, pkg)
	}
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
	json.NewEncoder(w).Encode(list)
}

func getPackage(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to get a package")
	id := r.URL.Query().Get("id")
	mu.Lock()
	defer mu.Unlock()
	if pkg, ok := packages[id]; ok {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
		json.NewEncoder(w).Encode(pkg)
	} else {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusNotFound)).Inc()
		http.NotFound(w, r)
	}
}

func updatePackageStatus(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to update package status")
	id := r.URL.Query().Get("id")
	status := r.URL.Query().Get("status")
	if status == "" {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusBadRequest)).Inc()
		http.Error(w, "Missing status", http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if pkg, ok := packages[id]; ok {
		pkg.Status = status
		packages[id] = pkg
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
		json.NewEncoder(w).Encode(pkg)
		slog.Info("Successfully updated package status", "id", pkg.ID, "status", status)
	} else {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusNotFound)).Inc()
		http.NotFound(w, r)
		slog.Warn("Package was not found", "id", id)
	}
}

func deletePackage(w http.ResponseWriter, r *http.Request) {
	requestsReceived.WithLabelValues(r.Method).Inc()
	slog.Info("Received request to delete package")
	if r.Method != http.MethodDelete {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusBadRequest)).Inc()
		http.Error(w, "Missing package id in delete request", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if _, ok := packages[id]; !ok {
		requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusNotFound)).Inc()
		http.NotFound(w, r)
		return
	}

	delete(packages, id)
	requestsProcessed.WithLabelValues(r.Method, strconv.Itoa(http.StatusOK)).Inc()
	w.WriteHeader(http.StatusOK)
}

func randomID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
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

	packageMux := http.NewServeMux()
	packageMux.HandleFunc("/packages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createPackage(w, r)
		} else {
			listPackages(w, r)
		}
	})
	packageMux.HandleFunc("/packages/get", getPackage)
	packageMux.HandleFunc("/packages/update", updatePackageStatus)
	packageMux.HandleFunc("/packages/delete", deletePackage)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		slog.Info("Prometheus metrics endpoint running", "addr", ":2112")
		if err := http.ListenAndServe(":2112", metricsMux); err != nil {
			slog.Error("metrics server error", "err", err)
			os.Exit(1)
		}
	}()

	slog.Info("PackageService running", "addr", ":8080")
	if err := http.ListenAndServe(":8080", packageMux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
