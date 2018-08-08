package myvar

import (
	"testing"
	"time"
)

func TestInt(t *testing.T) {
	SetDatabase("test")
	SetGlobalTag("global", "value")
	SetGlobalTag("name", "testone")
	if err := SetInfluxdb("http://localhost:8086"); err != nil {
		t.Fatal(err.Error())
	}
	SetFlushInterval(time.Second)
	n := NewInt("cpu", map[string]string{"host": "127.0.0.1"}, "value")
	n.Set(234)
	time.Sleep(time.Second * 2)
	n.Set(24)
	n.Set(241)
	time.Sleep(time.Second * 2)
	n.Set(24123)
	Flush(time.Now().Truncate(time.Second))

	f := NewFloat("cpu_idle", map[string]string{"host": "127.0.0.1"}, "value")
	f.Set(23.4)
	time.Sleep(time.Second * 2)
	f.Set(24.0)
	f.Set(24.1)
	time.Sleep(time.Second * 2)
	f.Set(241.23)
	Flush(time.Now().Truncate(time.Second))

	s := NewString("testinfo", map[string]string{"host": "127.0.0.1"}, "message")
	s.Set("hellladfadfa")
	time.Sleep(time.Second * 2)
	s.Set("second message")
	Flush(time.Now().Truncate(time.Second))

	s.Free()

}
