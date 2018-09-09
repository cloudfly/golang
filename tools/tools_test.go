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
