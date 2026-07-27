package jobs

import "testing"

func TestSignalCoalescesWakeHints(t *testing.T) {
	signal := NewSignal()
	for index := 0; index < 100; index++ {
		signal.Wake()
	}
	select {
	case <-signal.Notifications():
	default:
		t.Fatal("wake hint was lost")
	}
	select {
	case <-signal.Notifications():
		t.Fatal("wake hints were not coalesced")
	default:
	}
}
