package dconf

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDConf(t *testing.T) {
	conf1, err := NewDConf(context.Background(), []string{"http://localhost:2379"}, "/dconf_test")
	conf2, err := NewDConf(context.Background(), []string{"http://localhost:2379"}, "/dconf_test")
	assert.NoError(t, err)

	_, err = conf1.Get("hello")
	assert.True(t, IsKeyNotFound(err))
	assert.NoError(t, conf1.Set("hello", "world"))

	time.Sleep(time.Second / 2) // waiting for sync

	value, err := conf1.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, "world", value)

	value, err = conf2.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, "world", value)

	assert.NoError(t, conf1.Del("hello"))
	time.Sleep(time.Second / 2) // waiting for sync

	_, err = conf1.Get("hello")
	assert.True(t, IsKeyNotFound(err))

	_, err = conf2.Get("hello")
	assert.True(t, IsKeyNotFound(err))
}
