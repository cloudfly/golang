package myvar

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/influxdata/influxdb/models"
)

var (
	cacheLock sync.Mutex
	cache     map[string]*Var
)

func init() {
	cache = make(map[string]*Var)
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
