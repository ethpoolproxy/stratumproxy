package util

func StringSliceContain(slice []string, s string) bool {
	for _, s2 := range slice {
		if s == s2 {
			return true
		}
	}
	return false
}
