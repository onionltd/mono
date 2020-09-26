package evtmetrics

import (
	"context"
	"github.com/oniontree-org/go-oniontree/scanner"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	gauge *prometheus.GaugeVec
}

func (m *Metrics) ReadEvents(ctx context.Context, inputCh <-chan scanner.Event, outputCh chan<- scanner.Event) error {
	defer func() {
		if outputCh != nil {
			close(outputCh)
		}
	}()

	for {
		select {
		case event, more := <-inputCh:
			if !more {
				return nil
			}

			switch e := event.(type) {
			case scanner.ScanEvent:
				m.gauge.WithLabelValues(
					e.ServiceID,
					e.URL,
					e.Directory,
				).Set(float64(e.Status))
			}

			if outputCh != nil {
				outputCh <- event
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.gauge.Describe(ch)
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.gauge.Collect(ch)
}

func (m *Metrics) Get() *prometheus.GaugeVec {
	return m.gauge
}

func New() *Metrics {
	return &Metrics{
		gauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scanner_event_info",
				Help: "Scanner events.",
			},
			[]string{"service_id", "url", "directory"},
		),
	}
}
