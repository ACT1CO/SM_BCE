package main

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"
)

func now() string {
	return time.Now().Format("15:04")
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(strings.TrimPrefix(tag, "@"))
	tag = strings.ToLower(tag)
	var out []rune
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			out = append(out, r)
		}
	}
	return string(out)
}
