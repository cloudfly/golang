package myvar

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/Sirupsen/logrus"
	client "github.com/influxdata/influxdb/client/v2"
	"github.com/influxdata/influxdb/models"
)

var (
	flushInterval = time.Second * 10
	cacheLock     sync.Mutex
	cache         map[string]*Var
	tempCache     []*client.Point
	cancel        chan struct{}
	gtags         models.Tags

	database string
	c        client.Client
)

func init() {
	cache = make(map[string]*Var)
	cancel = make(chan struct{})
	tempCache = make([]*client.Point, 0, 1000)
	go flusher()
}

// SetDatabase initialize the default database
func SetDatabase(db string) {
	database = db
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

func publish(measurement string, tags map[string]string, name string, value interface{}) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	cache[key(measurement, tags, name)] = &Var{
		measurement: measurement,
		tags:        models.NewTags(tags),
		name:        name,
		value:       value,
	}
}

type Var struct {
	measurement string
	tags        models.Tags
	name        string
	value       interface{}
}

// Variable type
const (
	INT uint8 = 1 << iota
	FLOAT
	STRING
)

// Float variable
type Float struct {
	key   string
	value uint64
}

// NewFloat create a new float variable
func NewFloat(measurement string, tags map[string]string, name string) *Float {
	k := key(measurement, tags, name)
	cacheLock.Lock()
	data, ok := cache[k]
	cacheLock.Unlock()
	if ok {
		f, ok := data.value.(*Float)
		if ok {
			return f
		}
	}
	f := new(Float)
	f.key = k
	publish(measurement, tags, name, f)
	return f
}

func (f *Float) Set(v float64) {
	atomic.StoreUint64(&f.value, math.Float64bits(v))
}

func (f *Float) Add(delta float64) {
	for {
		cur := atomic.LoadUint64(&f.value)
		curVal := math.Float64frombits(cur)
		nxtVal := curVal + delta
		nxt := math.Float64bits(nxtVal)
		if atomic.CompareAndSwapUint64(&f.value, cur, nxt) {
			return
		}
	}
}

func (f *Float) Value() float64 {
	return math.Float64frombits(atomic.LoadUint64(&f.value))
}

func (f *Float) Free() {
	cacheLock.Lock()
	delete(cache, f.key)
	cacheLock.Unlock()
}

type Int struct {
	key   string
	value int64
}

func NewInt(measurement string, tags map[string]string, name string) *Int {
	k := key(measurement, tags, name)

	cacheLock.Lock()
	data, ok := cache[k]
	cacheLock.Unlock()
	if ok {
		n, ok := data.value.(*Int)
		if ok {
			return n
		}
	}
	n := new(Int)
	n.key = k
	publish(measurement, tags, name, n)
	return n
}

func (n *Int) Set(v int64) {
	atomic.StoreInt64(&(n.value), v)
}

func (n *Int) Add(v int64) {
	atomic.AddInt64(&(n.value), v)
}

func (n *Int) Incr() {
	atomic.AddInt64(&(n.value), 1)
}

func (n *Int) Value() int64 {
	return atomic.LoadInt64(&(n.value))
}

func (n *Int) Free() {
	cacheLock.Lock()
	delete(cache, n.key)
	cacheLock.Unlock()
}

type String struct {
	key   string
	value atomic.Value
}

func NewString(measurement string, tags map[string]string, name string) *String {
	k := key(measurement, tags, name)
	cacheLock.Lock()
	data, ok := cache[k]
	cacheLock.Unlock()
	if ok {
		s, ok := data.value.(*String)
		if ok {
			return s
		}
	}
	s := new(String)
	s.key = k
	publish(measurement, tags, name, s)
	return s
}

func (s *String) Set(v string) {
	if len(v) > 256 { // 不允许超过 256 长度
		v = v[:256]
	}
	s.value.Store(v)
}

func (s *String) Value() string {
	p, _ := s.value.Load().(string)
	return p
}

func (s *String) Free() {
	cacheLock.Lock()
	delete(cache, s.key)
	cacheLock.Unlock()
}

type Map struct {
	key  string
	data sync.Map
}

func NewMap(measurement string, tags map[string]string) *Map {
	k := key(measurement, tags, "")
	cacheLock.Lock()
	data, ok := cache[k]
	cacheLock.Unlock()
	if ok {
		m, ok := data.value.(*Map)
		if ok {
			return m
		}
	}
	m := new(Map)
	m.key = k
	publish(measurement, tags, "", m)
	return m
}

func (m *Map) Set(key string, value interface{}) *Map {
	m.data.Store(key, value)
	return m
}

func (m *Map) Get(key string) (interface{}, bool) {
	return m.data.Load(key)
}

func (m *Map) Free() {
	cacheLock.Lock()
	delete(cache, m.key)
	cacheLock.Unlock()
}

// Publish a raw influxdb points
func Publish(name string, tags map[string]string, fields map[string]interface{}) error {
	p, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		return err
	}
	cacheLock.Lock()
	defer cacheLock.Unlock()
	tempCache = append(tempCache, p)
	return nil
}

// Flush write all the points in cache into influxdb
func Flush(tt ...time.Time) error {
	var (
		t time.Time
	)
	if len(tt) > 0 {
		t = tt[0]
	} else {
		t = time.Now()
	}

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: database,
	})
	if err != nil {
		log.Errorf("fail to create influxdb batch, %s", err.Error())
		return err
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

	if len(tempCache) > 0 {
		batch.AddPoints(tempCache)
		tempCache = tempCache[:0]
	}
	cacheLock.Unlock()

	if len(batch.Points()) == 0 {
		return nil
	}

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

// 生成唯一 key, 必须要对 tags 排序, 否则相同的 metric 会出现不同的 key
func key(measurement string, tags map[string]string, name string) string {
	s := ""
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s += fmt.Sprintf("%s=%s,", k, tags[k])
	}
	return fmt.Sprintf("%s.%s.%s", measurement, s, name)
}
