package utils

import "regexp"

var urlRegex = regexp.MustCompile(`^https?://`)

func IsUrl(url string) bool {
	return urlRegex.MatchString(url)
}
