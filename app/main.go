package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Order struct {
	ID       int    `json:"id"`
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
}

var (
	mu     sync.Mutex
	orders = make(map[int]Order)
	nextID = 1

	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"path", "method", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"path"})
)

var failRate = 0.0

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func instrument(path string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		h(sw, r)
		requestDuration.WithLabelValues(path).Observe(time.Since(start).Seconds())
		requestsTotal.WithLabelValues(path, r.Method, strconv.Itoa(sw.status)).Inc()
	}
}

func maybeInjectFailure(w http.ResponseWriter) bool {
	if failRate <= 0 {
		return false
	}
	if rand.Float64() < failRate {
		time.Sleep(800 * time.Millisecond)
		http.Error(w, "simulated failure", http.StatusInternalServerError)
		return true
	}
	return false
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	if maybeInjectFailure(w) {
		return
	}
	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	mu.Lock()
	o.ID = nextID
	nextID++
	o.Status = "created"
	orders[o.ID] = o
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(o)
}

func listOrdersHandler(w http.ResponseWriter, r *http.Request) {
	if maybeInjectFailure(w) {
		return
	}
	mu.Lock()
	list := make([]Order, 0, len(orders))
	for _, o := range orders {
		list = append(list, o)
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func main() {
	if v := os.Getenv("FAIL_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			failRate = f
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/orders", instrument("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createOrderHandler(w, r)
		case http.MethodGet:
			listOrdersHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("orders-api listening on :8080, FAIL_RATE=%.2f", failRate)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
