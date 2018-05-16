package util

import "unicode"

func Ucfirst(s string) string {
	res := make([]rune, len(s))
	for i, r := range s {
		if i == 0 {
			res[i] = unicode.ToUpper(r)
		} else {
			res[i] = r
		}
	}
	return string(res)
}
