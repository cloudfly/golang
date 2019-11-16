package timepolicy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPolicy(t *testing.T) {
	// 2019-11-16 15:54:33
	initTime := time.Date(2019, 11, 16, 15, 54, 23, 0, time.Local)
	now := time.Date(2019, 11, 16, 15, 57, 41, 0, time.Local)
	p, err := ParsePolicy(initTime, "1m")
	assert.NoError(t, err)
	next := p.NextTime(now.Unix())
	assert.Equal(t, now.Truncate(time.Minute).Add(time.Minute).Unix(), next)

	p, _ = ParsePolicy(initTime, "1s")
	assert.Equal(t, now.Unix(), p.NextTime(now.Unix()))

	p, err = ParsePolicy(initTime, "1m:10s")
	assert.NoError(t, err)
	next = p.NextTime(now.Unix())
	assert.Equal(t, time.Date(2019, 11, 16, 15, 57, 43, 0, time.Local).Unix(), next)

	next = p.NextTime(initTime.Add(-time.Minute).Unix())
	assert.Equal(t, initTime.Add(time.Minute).Unix(), next) // next 即是开始时间

	p, err = ParsePolicy(initTime, "1m:10s:10m")
	assert.NoError(t, err)
	next = p.NextTime(now.Unix())
	assert.Equal(t, time.Date(2019, 11, 16, 15, 57, 43, 0, time.Local).Unix(), next)

	next = p.NextTime(initTime.Add(-time.Minute).Unix())
	assert.Equal(t, initTime.Add(time.Minute).Unix(), next) // next 即是开始时间

	next = p.NextTime(initTime.Add(time.Hour).Unix())
	assert.Equal(t, int64(0), next)

	p, err = ParsePolicy(initTime, "1m:10s:10m,10m:3m")
	assert.NoError(t, err)
	next = p.NextTime(now.Unix())
	assert.Equal(t, time.Date(2019, 11, 16, 15, 57, 43, 0, time.Local).Unix(), next)

	next = p.NextTime(initTime.Add(-time.Minute).Unix())
	assert.Equal(t, initTime.Add(time.Minute).Unix(), next) // next 即是开始时间

	next = p.NextTime(now.Add(time.Hour).Unix())
	// 走 10m:3m 策略, 2019-11-16 16:04:23 开始每 3m 一次
	// now.Add(time.Hour) 是否 2019-11-16 16:57:41
	// 16:04:23 + 3m * 18 => 16:04:23 + 54m => 16:58:23
	assert.Equal(t, time.Date(2019, 11, 16, 16, 58, 23, 0, time.Local).Unix(), next)
}

type MockJob struct {
	t        *testing.T
	from     time.Time
	finished bool
}

func (job *MockJob) Do(t time.Time) {
	fmt.Println(t.Format("2006-01-02T15:04:05"))
}

func (job *MockJob) Finished() bool {
	return job.finished
}

func TestEngine(t *testing.T) {
	engine := NewEngine(context.Background())

	job := &MockJob{
		from: time.Now(),
		t:    t,
	}
	fmt.Println("start time is:", job.from.Format("2006-01-02T15:04:05"))
	err := engine.RegisterWithTime(job.from, ":2s:10s,20s:5s:10m", job)
	assert.NoError(t, err)
	time.Sleep(time.Second * 30)
	job.finished = true
	fmt.Println("finished, output nothing from now")
	time.Sleep(time.Second * 10)
}
