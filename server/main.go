package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var httpClient *http.Client

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	initKubernetesClient()

	http.HandleFunc("/api/ingresses", handleIngresses)

	fs := http.FileServer(http.Dir("/app"))
	http.Handle("/", fs)

	log.Printf("Serving on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func initKubernetesClient() {
	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Printf("Warning: Could not read CA cert: %v (running outside cluster?)", err)
		httpClient = &http.Client{Timeout: 10 * time.Second}
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	httpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
}

func handleIngresses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ingresses, err := fetchIngresses()
	if err != nil {
		log.Printf("Error fetching ingresses: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(ingresses)
}

func fetchIngresses() (map[string]interface{}, error) {
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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
