package helpers

func FormValueTrue(v string) bool {
	if v == "yes" || v == "on" || v == "1" || v == "true" {
		return true
	}

	return false
}
