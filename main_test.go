package main

import (
	"testing"
	"time"
)

func TestFormatTimeInDays(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		t      time.Time
		format string
	}{
		{now, "Today"},
		{now.Add(24 * time.Hour), "Tomorrow"},
		{now.Add(-24 * time.Hour), "Yesterday"},
		{now.Add(7 * -24 * time.Hour), "7 Days ago"},
		{now.Add(7 * 24 * time.Hour), "In 7 Days"},
	}

	for _, tc := range testCases {
		res := formatTimeInDays(tc.t)
		if res != tc.format {
			t.Fatalf("formatted string not correct. expected=%q, got=%q", tc.format, res)
		}
	}
}
