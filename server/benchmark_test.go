package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkFetchIngresses(b *testing.B) {
	kubernetesServiceHost = ""
	kubernetesServicePort = ""

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fetchIngresses(ctx)
	}
}

func BenchmarkIsReady_NoIO(b *testing.B) {
	// Outside Kubernetes case
	kubernetesServiceHost = ""
	kubernetesServicePort = ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isReady()
	}
}

func BenchmarkIsReady_WithIO(b *testing.B) {
	// In-cluster case (simulated)
	kubernetesServiceHost = "10.0.0.1"
	kubernetesServicePort = "443"
	httpClient = &http.Client{}

	tmpDir, err := os.MkdirTemp("", "hp-bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origTokenPath := tokenPath
	tokenPath = filepath.Join(tmpDir, "token")
	defer func() { tokenPath = origTokenPath }()

	err = os.WriteFile(tokenPath, []byte("some-token-value"), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isReady()
	}
}
