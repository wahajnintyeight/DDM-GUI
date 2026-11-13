package main

import (
	"encoding/json"
	"strings"
	"time"
)

const metricsPrefix = "DEPOTDROP_METRICS "

type JobMetrics struct {
	DownloadedBytes int64   `json:"downloadedBytes"`
	ValidationBytes int64   `json:"validationBytes"`
	DownloadRate    float64 `json:"downloadRate"`
	ValidationRate  float64 `json:"validationRate"`
}

type metricsSampler struct {
	previous JobMetrics
	at       time.Time
}

func (s *metricsSampler) parse(line string, now time.Time) (*JobMetrics, bool) {
	if !strings.HasPrefix(line, metricsPrefix) {
		return nil, false
	}
	var next JobMetrics
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, metricsPrefix)), &next); err != nil || next.DownloadedBytes < 0 || next.ValidationBytes < 0 {
		return nil, true
	}
	if !s.at.IsZero() && next.DownloadedBytes >= s.previous.DownloadedBytes && next.ValidationBytes >= s.previous.ValidationBytes {
		seconds := now.Sub(s.at).Seconds()
		if seconds > 0 {
			next.DownloadRate = float64(next.DownloadedBytes-s.previous.DownloadedBytes) / seconds
			next.ValidationRate = float64(next.ValidationBytes-s.previous.ValidationBytes) / seconds
		}
	}
	s.previous, s.at = next, now
	return &next, true
}
