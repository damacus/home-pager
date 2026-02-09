package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var httpClient *http.Client

const (
	defaultPort           = "8080"
	defaultHTTPTimeout    = 10 * time.Second
	maxIngressesBodyBytes = 4 << 20
)

var startTime = time.Now()
var totalRequests uint64

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	kubeTimeout := getEnvDuration("KUBERNETES_TIMEOUT", defaultHTTPTimeout)
	initKubernetesClient(kubeTimeout)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ingresses", handleIngresses(kubeTimeout))
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/readyz", handleReady)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.Handle("/", http.FileServer(http.Dir("/app")))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           withSecurityHeaders(withRequestMetrics(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownErr := make(chan error, 1)
	go func() {
		log.Printf("Serving on :%s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr <- err
		}
		close(shutdownErr)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-shutdownErr:
		if err != nil {
			log.Fatal(err)
		}
	case <-stop:
		log.Printf("Shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

func initKubernetesClient(timeout time.Duration) {
	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Printf("Warning: Could not read CA cert: %v (running outside cluster?)", err)
		httpClient = &http.Client{Timeout: timeout}
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	httpClient = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
}

func handleIngresses(timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		ingresses, err := fetchIngresses(ctx)
		if err != nil {
			log.Printf("Error fetching ingresses: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(ingresses)
	}
}

func fetchIngresses(ctx context.Context) (map[string]interface{}, error) {
	apiServer := os.Getenv("KUBERNETES_SERVICE_HOST")
	apiPort := os.Getenv("KUBERNETES_SERVICE_PORT")

	if apiServer == "" || apiPort == "" {
		return map[string]interface{}{"items": []interface{}{}}, nil
	}

	tokenBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(string(tokenBytes))

	url := "https://" + apiServer + ":" + apiPort + "/apis/networking.k8s.io/v1/ingresses"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxIngressesBodyBytes))
		return nil, errors.New("kubernetes api error: " + resp.Status + " " + strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxIngressesBodyBytes))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleReady(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !isReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	uptime := time.Since(startTime).Seconds()
	requests := atomic.LoadUint64(&totalRequests)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "# HELP home_pager_uptime_seconds Process uptime in seconds.\n")
	_, _ = io.WriteString(w, "# TYPE home_pager_uptime_seconds gauge\n")
	_, _ = io.WriteString(w, "home_pager_uptime_seconds ")
	_, _ = io.WriteString(w, strconv.FormatFloat(uptime, 'f', 0, 64))
	_, _ = io.WriteString(w, "\n")
	_, _ = io.WriteString(w, "# HELP home_pager_http_requests_total Total HTTP requests served.\n")
	_, _ = io.WriteString(w, "# TYPE home_pager_http_requests_total counter\n")
	_, _ = io.WriteString(w, "home_pager_http_requests_total ")
	_, _ = io.WriteString(w, strconv.FormatUint(requests, 10))
	_, _ = io.WriteString(w, "\n")
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func withRequestMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&totalRequests, 1)
		next.ServeHTTP(w, r)
	})
}

func getEnvDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return fallback
		}
		return time.Duration(seconds) * time.Second
	}

	if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
		return parsed
	}

	return fallback
}

func isReady() bool {
	apiServer := strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST"))
	apiPort := strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_PORT"))

	// Outside Kubernetes, always report ready for local/dev usage.
	if apiServer == "" || apiPort == "" {
		return true
	}

	if httpClient == nil {
		return false
	}

	tokenBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(tokenBytes)) != ""
}
