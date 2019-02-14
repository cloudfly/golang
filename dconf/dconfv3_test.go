package dconf

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDConf3(t *testing.T) {
	conf1, err := NewV3(context.Background(), []string{"localhost:2379"}, "/dconf_test")
	assert.NoError(t, err)
	conf2, err := NewV3(context.Background(), []string{"localhost:2379"}, "/dconf_test")
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

	data := conf1.Data("")
	assert.Equal(t, 2, len(data))
	assert.Equal(t, "readonly", data["database/table1"])
	assert.Equal(t, "writeonly", data["database/table2"])

	conf3, err := NewV3(context.Background(), []string{"http://localhost:2379"}, "/dconf_test")
	assert.NoError(t, err)

	data = conf3.Data("")
	assert.Equal(t, 2, len(data))
	assert.Equal(t, "readonly", data["database/table1"])
	assert.Equal(t, "writeonly", data["database/table2"])

	assert.NoError(t, conf3.Set("hello", "world"))

	time.Sleep(time.Second / 2)

	data = conf1.Data("database")
	assert.Equal(t, 2, len(data))
	assert.Equal(t, "readonly", data["database/table1"])
	assert.Equal(t, "writeonly", data["database/table2"])

	conf1.data.Range(func(key, value interface{}) bool {
		assert.NoError(t, conf1.Del(key.(string)))
		return true
	})
}
