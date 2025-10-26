package services

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	totalFrames   atomic.Int64
	totalErrors   atomic.Int64
	totalLatency  atomic.Int64
	activeClients atomic.Int32
	lastFrameTime atomic.Int64
}

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

func NewMetrics() *Metrics {
	return &Metrics{}
}

func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = NewMetrics()
	})
	return metricsInstance
}

func (m *Metrics) IncrementFrames() {
	m.totalFrames.Add(1)
	m.lastFrameTime.Store(time.Now().Unix())
}

func (m *Metrics) IncrementErrors() {
	m.totalErrors.Add(1)
}

func (m *Metrics) RecordLatency(duration time.Duration) {
	m.totalLatency.Add(duration.Milliseconds())
}

func (m *Metrics) SetActiveClients(count int) {
	m.activeClients.Store(int32(count))
}

func (m *Metrics) GetTotalFrames() int64 {
	return m.totalFrames.Load()
}

func (m *Metrics) GetTotalErrors() int64 {
	return m.totalErrors.Load()
}

func (m *Metrics) GetAvgLatency() float64 {
	frames := m.totalFrames.Load()
	if frames == 0 {
		return 0
	}
	return float64(m.totalLatency.Load()) / float64(frames)
}

func (m *Metrics) GetActiveClients() int {
	return int(m.activeClients.Load())
}

func (m *Metrics) GetLastFrameTime() int64 {
	return m.lastFrameTime.Load()
}
