package main

import (
	"time"
)

// chkStringValue checks if a pointer to a string is nil and ifso
// it returns a pointer to an empty string
func chkStringValue(s *string) *string {
	if s == nil {
		emptyString := ""
		s = &emptyString
	}
	return s
}

// chkTimeValue checks if a pointer to a time struct is nil and if so
// it returns a pointer to an empty struct
func chkTimeValue(t *time.Time) *time.Time {
	if t == nil {
		return &time.Time{}
	}
	return t
}

func safeDate(t *time.Time) string {
	if zero := chkTimeValue(t).IsZero(); zero == true {
		return ""
	} else {
		return t.String()
	}
}

/*

*/
