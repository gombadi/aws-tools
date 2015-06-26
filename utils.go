package main

// chkStringValue checks if a pointer to a string is nil and ifso
// it returns a pointer to an empty string
func chkStringValue(s *string) *string {
	if s == nil {
		emptyString := ""
		s = &emptyString
	}
	return s
}
