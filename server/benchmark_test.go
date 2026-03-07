package main

import (
	"context"
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

func BenchmarkIsReady(b *testing.B) {
	kubernetesServiceHost = ""
	kubernetesServicePort = ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isReady()
	}
}
