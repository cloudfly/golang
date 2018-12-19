package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	h1 := Hash([]byte("hello"))
	h2 := Hash([]byte("hello"))
	assert.Equal(t, h1, h2)
	h3 := Hash([]byte("hello "))
	assert.NotEqual(t, h1, h3)
}

func TestSimpleMatch(t *testing.T) {
	assert.True(t, SimpleMatch("*", "abc"))
	assert.True(t, SimpleMatch("*hello", "hello"))
	assert.True(t, SimpleMatch("*hello", "gogohello"))
	assert.True(t, SimpleMatch("*hello*", "gogohello world"))
	assert.True(t, SimpleMatch("**", "gogohello world"))
	assert.False(t, SimpleMatch("*p*", "world"))
	assert.True(t, SimpleMatch("*", ""))
	assert.True(t, SimpleMatch("", ""))
	assert.False(t, SimpleMatch("abc", ""))
	assert.True(t, SimpleMatch("abc", "abc"))
	assert.False(t, SimpleMatch("abc", "abcd"))
	assert.False(t, SimpleMatch("abcd", "ab"))

	assert.True(t, SimpleMatch("he*wor*", "hello world"))
	assert.True(t, SimpleMatch("he*wor*d", "hello world"))
	assert.False(t, SimpleMatch("he*you", "hello world"))
}
