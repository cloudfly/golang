package dconf

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDConf(t *testing.T) {
	conf1, err := New(context.Background(), []string{"http://localhost:2379"}, "/dconf_test")
	conf2, err := New(context.Background(), []string{"http://localhost:2379"}, "/dconf_test")
	assert.NoError(t, err)

	_, err = conf1.Get("hello")
	assert.True(t, IsKeyNotFound(err))
	assert.NoError(t, conf1.Set("hello", "world"))

	time.Sleep(time.Second) // waiting for sync

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

	assert.NoError(t, conf1.Set("database/table1", "readonly"))
	assert.NoError(t, conf1.Set("database/table2", "writeonly"))
	time.Sleep(time.Second / 2) // waiting for sync

	keys := conf1.Keys()
	assert.Equal(t, 2, len(keys))

	data := conf1.Data()
	assert.Equal(t, 2, len(data))
	assert.Equal(t, "readonly", data["database/table1"])
	assert.Equal(t, "writeonly", data["database/table2"])

	conf3, err := New(context.Background(), []string{"http://localhost:2379"}, "/dconf_test")
	assert.NoError(t, err)

	data = conf3.Data()
	assert.Equal(t, 2, len(data))
	assert.Equal(t, "readonly", data["database/table1"])
	assert.Equal(t, "writeonly", data["database/table2"])
}
