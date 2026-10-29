package main

import (
	"strings"
	"unicode/utf8"
)

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if x := strings.TrimSpace(l); x != "" {
			if utf8.RuneCountInString(x) > 60 {
				x = string([]rune(x)[:57]) + "…"
			}
			return x
		}
	}
	return ""
}

func displayFolder(s string) string {
	if s == "" {
		return "Root"
	}
	return s
}
