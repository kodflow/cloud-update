package logger

import (
	"testing"
)

// BenchmarkLogrus tests logrus performance.
func BenchmarkLogrus(b *testing.B) {
	_ = Initialize(Config{
		Level:    "info",
		FilePath: "", // No file, just stdout
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithFields(map[string]interface{}{
			"request_id": i,
			"user_id":    "test",
			"action":     "benchmark",
		}).Info("Benchmark message")
	}
}

// BenchmarkSlog tests slog performance.
func BenchmarkSlog(b *testing.B) {
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: "", // No file, just stdout
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogInfo("Benchmark message",
			"request_id", i,
			"user_id", "test",
			"action", "benchmark",
		)
	}
}

// BenchmarkLogrusParallel tests logrus with concurrent writes.
func BenchmarkLogrusParallel(b *testing.B) {
	_ = Initialize(Config{
		Level:    "info",
		FilePath: "",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			WithFields(map[string]interface{}{
				"request_id": i,
				"user_id":    "test",
				"action":     "benchmark",
			}).Info("Parallel benchmark")
			i++
		}
	})
}

// BenchmarkSlogParallel tests slog with concurrent writes.
func BenchmarkSlogParallel(b *testing.B) {
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: "",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			LogInfo("Parallel benchmark",
				"request_id", i,
				"user_id", "test",
				"action", "benchmark",
			)
			i++
		}
	})
}
