package myvar

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/Sirupsen/logrus"
	client "github.com/influxdata/influxdb/client/v2"
	"github.com/influxdata/influxdb/models"
)

var (
	metricID      int64
	flushInterval = time.Second * 10
	metricCache   sync.Map
	cancel        chan struct{}
	gtags         models.Tags

	batch    client.BatchPoints
	database string
	rp       string
	c        client.Client
)

func init() {
	cache = make(map[string]*Var)
	cancel = make(chan struct{})
	go flusher()
}

// SetDatabase initialize the default database
func SetDatabase(db string, rpSetting ...string) {
	database = db
	if len(rpSetting) > 0 {
		rp = rpSetting[0]
	}
	resetBatch()
}

func GetDatabase() string {
	return database
}

// SetInfluxdb initialize the influxdb address
func SetInfluxdb(addr string) (err error) {
	c, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:    addr,
		Timeout: time.Second * 10,
	})
	return
}

// SetFlushInterval change the flush interval
func SetFlushInterval(dur time.Duration) {
	flushInterval = dur
	close(cancel)
	cancel = make(chan struct{})
	go flusher()
}

// SetGlobalTag set global tag list, used by each variable
func SetGlobalTag(key, value string) {
	gtags.SetString(key, value)
}

func resetBatch() {
	batch, _ = client.NewBatchPoints(client.BatchPointsConfig{
		Database:        database,
		RetentionPolicy: rp,
	})
}

type Metric struct {
	*sync.RWMutex
	id          int64
	measurement string
	tags        map[string]*string
	ints        map[string]int64
	floats      map[string]float64
	strs        map[string]string
}

func NewMetric(measurement string, tags map[string]*string) *Metric {
	if tags == nil {
		tags = make(map[string]*string)
	}
	metric := &Metric{
		RWMutex:     &sync.RWMutex{},
		measurement: measurement,
		id:          atomic.AddInt64(&metricID, 1),
		tags:        tags,
		ints:        make(map[string]int64),
		floats:      make(map[string]float64),
		strs:        make(map[string]string),
	}
	metricCache.Store(metric.id, metric)
	return metric
}

func (metric *Metric) Free() {
	metricCache.Delete(metric.id)
}

func (metric *Metric) Incr(field string) *Metric {
	metric.Lock()
	defer metric.Unlock()
	i, _ := metric.ints[field]
	metric.ints[field] = i + 1
	return metric
}

func (metric *Metric) Add(field string, v int64) *Metric {
	metric.Lock()
	defer metric.Unlock()
	i, _ := metric.ints[field]
	metric.ints[field] = i + v
	return metric
}

func (metric *Metric) Set(field string, v int64) *Metric {
	metric.Lock()
	defer metric.Unlock()
	metric.ints[field] = v
	return metric
}

func (metric *Metric) Clear(field string) *Metric {
	metric.Lock()
	defer metric.Unlock()
	delete(metric.ints, field)
	delete(metric.floats, field)
	delete(metric.strs, field)
	return metric
}

func (metric *Metric) AddFloat(field string, v float64) *Metric {
	metric.Lock()
	defer metric.Unlock()
	f, _ := metric.floats[field]
	metric.floats[field] = f + v
	return metric
}

func (metric *Metric) SetFloat(field string, v float64) *Metric {
	metric.Lock()
	defer metric.Unlock()
	metric.floats[field] = v
	return metric
}

func (metric *Metric) SetString(field string, str string) *Metric {
	metric.Lock()
	defer metric.Unlock()
	metric.strs[field] = str
	return metric
}

func (metric *Metric) Append(field string, str string) *Metric {
	metric.Lock()
	defer metric.Unlock()
	s, _ := metric.strs[field]
	if len(s) >= 512 {
		return metric
	}
	metric.strs[field] = s + str
	return metric
}

func (metric *Metric) AppendNoLimit(field string, str string) *Metric {
	metric.Lock()
	defer metric.Unlock()
	s, _ := metric.strs[field]
	metric.strs[field] = s + str
	return metric
}

func (metric *Metric) Int(field string) int64 {
	metric.RLock()
	defer metric.RUnlock()
	i, _ := metric.ints[field]
	return i
}

func (metric *Metric) Float(field string) float64 {
	metric.RLock()
	defer metric.RUnlock()
	f, _ := metric.floats[field]
	return f
}

func (metric *Metric) String(field string) string {
	metric.RLock()
	defer metric.RUnlock()
	s, _ := metric.strs[field]
	return s
}

func (metric *Metric) Points(t time.Time) *client.Point {
	metric.RLock()
	defer metric.RUnlock()
	fields := make(map[string]interface{})
	tags := make(map[string]string)
	for k, v := range metric.tags {
		k, v := k, v
		tags[k] = *v
	}
	for k, v := range metric.ints {
		k, v := k, v
		fields[k] = v
	}
	for k, v := range metric.floats {
		k, v := k, v
		fields[k] = v
	}
	for k, v := range metric.strs {
		k, v := k, v
		fields[k] = v
		delete(metric.strs, k)
	}

	p, err := models.NewPoint(metric.measurement, append(models.NewTags(tags), gtags...), models.Fields(fields), t)
	if err != nil {
		log.Errorf("failed create new influxdb point, %s", err.Error())
		return nil
	}
	return client.NewPointFrom(p)
}

// Publish a raw influxdb points
func Publish(name string, tags map[string]string, fields map[string]interface{}) error {
	p, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		return err
	}
	batch.AddPoint(p)
	return nil
}

// Flush write all the points in cache into influxdb
func Flush(tt ...time.Time) error {
	t := time.Now()
	if len(tt) > 0 {
		t = tt[0]
	}

	cacheLock.Lock()
LOOP:
	for _, v := range cache {
		fields := map[string]interface{}{}
		switch vv := v.value.(type) {
		case *Float:
			fields[v.name] = vv.Value()
		case *Int:
			fields[v.name] = vv.Value()
		case *String:
			s := vv.Value()
			if len(s) == 0 {
				continue LOOP
			}
			fields[v.name] = s
			vv.Set("") // 重置一下, 下次除非有新的数据过来, 否则就不发了
		case *Map:
			vv.data.Range(func(k, v interface{}) bool {
				fields[k.(string)] = v
				vv.data.Delete(k)
				return true
			})
		}
		if len(fields) == 0 {
			continue
		}
		p, err := models.NewPoint(v.measurement, append(v.tags.Clone(), gtags...), models.Fields(fields), t)
		if err != nil {
			log.Errorf("failed create new influxdb point, %s", err.Error())
			continue
		}
		batch.AddPoint(client.NewPointFrom(p))
	}
	cacheLock.Unlock()

	metricCache.Range(func(_, value interface{}) bool {
		metric := value.(*Metric)
		if p := metric.Points(t); p != nil {
			batch.AddPoint(p)
		}
		return true
	})

	if len(batch.Points()) == 0 {
		return nil
	}

	defer resetBatch()

	return c.Write(batch)
}

func flusher() {
	ticker := time.Tick(flushInterval)
	for {
		select {
		case t := <-ticker:
			if database == "" || c == nil {
				break
			}
			if err := Flush(t.Truncate(time.Second)); err != nil {
				log.Errorf("failed to flush points into influxdb, %s", err.Error())
			}
		case <-cancel:
			if err := Flush(time.Now().Truncate(time.Second)); err != nil {
				log.Errorf("failed to flush points into influxdb, %s", err.Error())
			}
			return
		}
	}
}

func key(measurement string, tags map[string]string, name string) string {
	s := ""
	for k, v := range tags {
		s += fmt.Sprintf("%s=%s,", k, v)
	}
	return fmt.Sprintf("%s.%s.%s", measurement, s, name)
}
