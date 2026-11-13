package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestTelemetryProcessHelper(t *testing.T) {
	if os.Getenv("DEPOTDROP_METRICS_HELPER") != "1" {
		return
	}
	fmt.Println(`DEPOTDROP_METRICS {"downloadedBytes":0,"validationBytes":0}`)
	time.Sleep(20 * time.Millisecond)
	fmt.Println(`DEPOTDROP_METRICS {"downloadedBytes":1200,"validationBytes":2400}`)
	os.Exit(0)
}

func TestDownloadAttemptEmitsJobMetrics(t *testing.T) {
	t.Setenv("DEPOTDROP_METRICS_HELPER", "1")
	var samples []*JobMetrics
	result := downloadAttempt(context.Background(), os.Args[0], []string{"-test.run=^TestTelemetryProcessHelper$"}, "", func(update DownloadUpdate) {
		if update.Metrics != nil {
			samples = append(samples, update.Metrics)
		}
		if update.Line != "" && update.Line[:min(len(update.Line), len(metricsPrefix))] == metricsPrefix {
			t.Fatal("telemetry leaked into the log")
		}
	})
	if result.State != "complete" || len(samples) != 2 || samples[1].DownloadedBytes != 1200 || samples[1].ValidationBytes != 2400 || samples[1].DownloadRate <= 0 {
		t.Fatalf("result %s, samples %+v", result.State, samples)
	}
}

func TestMetricsSampler(t *testing.T) {
	var sampler metricsSampler
	start := time.Unix(100, 0)
	first, ok := sampler.parse(`DEPOTDROP_METRICS {"downloadedBytes":100,"validationBytes":300}`, start)
	if !ok || first == nil || first.DownloadRate != 0 {
		t.Fatalf("initial sample: %+v", first)
	}
	next, ok := sampler.parse(`DEPOTDROP_METRICS {"downloadedBytes":300,"validationBytes":900}`, start.Add(2*time.Second))
	if !ok || next == nil || next.DownloadRate != 100 || next.ValidationRate != 300 {
		t.Fatalf("rates: %+v", next)
	}
	if sample, ok := sampler.parse(`DEPOTDROP_METRICS {bad}`, start); !ok || sample != nil {
		t.Fatal("invalid telemetry accepted")
	}
	if _, ok := sampler.parse("ordinary log line", start); ok {
		t.Fatal("ordinary log consumed")
	}
}
