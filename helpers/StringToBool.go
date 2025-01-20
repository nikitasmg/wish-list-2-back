package helpers

func StringToBool(s string) bool {
	if s == "true" || s == "1" {
		return true
	} else {
		return false
	}
}
