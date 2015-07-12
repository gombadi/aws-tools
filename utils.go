package main

import (
	"time"
)

// safeString checks if it is passed a nil pointer and if so returns an empty
// string else returns the string the pointer is pointing to
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// safeDate checks if a *time.Time is nil and if so returns
// an string of the zero time.Time
func safeDateString(t *time.Time) string {
	if t == nil {
		return time.Time{}.String()
	}
	return t.String()
}

/*

 */
