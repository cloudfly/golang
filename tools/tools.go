package tools

import (
	"bytes"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
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

func RSAPublickDecryptEasy(key, data []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %s", err.Error())
	}
	return RSAPublickDecrypt(pub.(*rsa.PublicKey), data), nil
}

func RSAPublickDecrypt(pubKey *rsa.PublicKey, data []byte) []byte {
	c := new(big.Int)
	m := new(big.Int)
	m.SetBytes(data)
	e := big.NewInt(int64(pubKey.E))
	c.Exp(m, e, pubKey.N)
	out := c.Bytes()
	skip := 0
	for i := 2; i < len(out); i++ {
		if i+1 >= len(out) {
			break
		}
		if out[i] == 0xff && out[i+1] == 0 {
			skip = i + 2
			break
		}
	}
	return out[skip:]
}
