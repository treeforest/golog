package golog

import (
	"context"
	"io"
	"testing"
)

func BenchmarkInfo_Discard(b *testing.B) {
	cfg := NewConfig(
		WithLogInConsole(false),
		WithLogInFile(false),
	)
	l, err := NewLogger(cfg, io.Discard)
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark log message")
	}
}

func BenchmarkInfow_Structured(b *testing.B) {
	cfg := NewConfig(
		WithLogInConsole(false),
		WithLogInFile(false),
	)
	l, err := NewLogger(cfg, io.Discard)
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Infow("structured benchmark",
			"key1", "value1",
			"key2", 12345,
			"key3", true,
			"key4", "value4",
		)
	}
}

func BenchmarkContext_InfoCtx(b *testing.B) {
	cfg := NewConfig(
		WithLogInConsole(false),
		WithLogInFile(false),
	)
	l, err := NewLogger(cfg, io.Discard)
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()

	ctx := ContextWithLogger(context.Background(), l)
	ctx = ContextWithTrace(ctx, "trace-bench-123456", "req-bench-789012")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InfowCtx(ctx, "context message", "status", 200)
	}
}

func BenchmarkGlobal_Parallel(b *testing.B) {
	cfg := NewConfig(
		WithLogInConsole(false),
		WithLogInFile(false),
	)
	l := MustNewLogger(cfg, io.Discard)
	SetDefaultLogger(l)
	defer Close()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("parallel log message")
		}
	})
}
