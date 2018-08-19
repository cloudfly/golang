package funcs

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
)

func Hash(data []byte) string {
	s := sha1.Sum(data)
	m := md5.Sum(data)
	return fmt.Sprintf("%x%x", s[:8], m[:8])
}
