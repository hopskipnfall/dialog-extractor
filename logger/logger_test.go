package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	target := New()

	target.Print("Hey you!")

	if want, got := "Hey you!", target.logOutput; want != got {
		t.Errorf("logOutput = %v, want %v", got, want)
	}
}
