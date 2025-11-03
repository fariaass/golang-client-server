package metrics

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 1. Contador de Requisições
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_server_http_requests_total",
			Help: "Total de requisições HTTP recebidas.",
		},
		[]string{"handler", "method", "code"}, // Labels
	)

	// 2. Histograma de Duração das Requisições
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "go_server_http_request_duration_seconds",
			Help: "Duração (latência) das requisições HTTP em segundos.",
			// Buckets (faixas) para o histograma. Pode ajustar conforme necessário.
			Buckets: prometheus.DefBuckets, 
		},
		[]string{"handler", "method"}, // Labels
	)
)

// --- Middleware (Definido no Passo 3) ---
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{w, http.StatusOK}
}

func (srw *statusResponseWriter) WriteHeader(code int) {
	srw.statusCode = code
	srw.ResponseWriter.WriteHeader(code)
}

func PrometheusMiddleware(next http.Handler, handlerLabel string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		startTime := time.Now()
		srw := newStatusResponseWriter(w)

		next.ServeHTTP(srw, r)

		duration := time.Since(startTime)
		log.Printf("Request handled: Method=%s, Path=%s, Latency=%s", r.Method, r.URL.Path, duration)
		method := r.Method
		code := strconv.Itoa(srw.statusCode)

		httpRequestsTotal.WithLabelValues(handlerLabel, method, code).Inc()
		httpRequestDuration.WithLabelValues(handlerLabel, method).Observe(duration.Seconds())
	})
}
