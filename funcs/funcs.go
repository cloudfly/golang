package funcs

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
)

func Hash(data []byte) string {
	s := sha1.Sum(data)
	m := md5.Sum(data)
	return fmt.Sprintf("%x%x", s[:8], m[:8])
}

func SimpleMatch(pattern, s string) bool {
	if pattern == "" { // empty pattern only match empty string
		return s == ""
	}
	patternBytes, name := []byte(pattern), []byte(s) // parse to []byte saving the memroy and reduce gc for string
	items := bytes.Split(patternBytes, []byte{'*'})
	for i, item := range items {
		if i == len(items)-1 && len(item) == 0 { // pattern end with *
			return true
		}
		j := bytes.Index(name, item)
		if j == -1 {
			return false
		}
		if i == 0 && len(item) != 0 && j != 0 { // 保证 abc* 匹配以 abc 开头的, 否则会匹配到 aabcxx 这种
			return false
		}
		name = name[j+len(item):]
	}
	return len(name) == 0
}
