package myvar_class

import (
	"testing"
	"time"
	log "github.com/Sirupsen/logrus"
)

func TestInt(t *testing.T) {
	mv, err := NewMyVar("http://localhost:8087", "myvar_monitor", time.Second*3)
	if err != nil {
		log.Fatal(err)
	}

	mv.SetGlobalTag("global", "value23")
	mv.SetGlobalTag("name", "testone")
	n := mv.NewInt("cpu", map[string]string{"host": "127.0.0.1"}, "value")
	n.Set(234)
	time.Sleep(time.Second)
	n.Set(24)
	n.Set(241)
	time.Sleep(time.Second)
	n.Set(24123)
	mv.Flush(time.Now().Truncate(time.Second))
	mv.FreeInt(n)

	f := mv.NewFloat("cpu_idle", map[string]string{"host": "127.0.0.1"}, "value")
	f.Set(23.4)
	time.Sleep(time.Second)
	f.Set(24.0)
	f.Set(24.1)
	time.Sleep(time.Second)
	f.Set(241.23)
	mv.Flush(time.Now().Truncate(time.Second))

	mv.FreeFloat(f)

	s := mv.NewString("testinfo", map[string]string{"host": "127.0.0.1"}, "message")
	s.Set("hellladfadfa")
	time.Sleep(time.Second * 2)
	s.Set("second message")
	mv.Flush(time.Now().Truncate(time.Second))
	mv.FreeString(s)

	mv.NewMap("testinfo", map[string]string{"host": "127.0.0.2"})

}
