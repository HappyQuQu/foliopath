// Package jobs owns process-local wake-up hints for durable database queues.
// Signals never carry work and are never a source of correctness.
package jobs

type Signal struct {
	channel chan struct{}
}

func NewSignal() *Signal {
	return &Signal{channel: make(chan struct{}, 1)}
}

func (signal *Signal) Wake() {
	select {
	case signal.channel <- struct{}{}:
	default:
	}
}

func (signal *Signal) Notifications() <-chan struct{} {
	return signal.channel
}
