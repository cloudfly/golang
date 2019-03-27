package ticketbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBox(t *testing.T) {
	n := 0
	for i := 0; i < 60; i++ {
		// 1分钟内最多发 10 票
		if _, err := Get("test", 10, 60); err == nil {
			n++
		}
	}
	assert.Equal(t, 10, n)
}
