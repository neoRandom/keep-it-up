package driven

import (
	"testing"
	"time"
)

func TestDefaultTimeProvider_Time(t *testing.T) {
	before := time.Now()
	got, err := (DefaultTimeProvider{}).Time()
	after := time.Now()
	if err != nil {
		t.Fatalf("Time() returned error: %v", err)
	}
	if got.Before(before) || got.After(after) {
		t.Errorf("Time() = %v, want a time between %v and %v", got, before, after)
	}
}