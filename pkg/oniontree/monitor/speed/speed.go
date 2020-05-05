package speed

import "time"

type Measurement struct {
	start   time.Time
	average time.Duration
}

func (m *Measurement) Start() {
	m.start = time.Now()
}

func (m *Measurement) Stop() time.Duration {
	d := time.Since(m.start)
	m.average = (m.average + d) / 2
	return d
}

func (m Measurement) Average() time.Duration {
	return m.average
}
