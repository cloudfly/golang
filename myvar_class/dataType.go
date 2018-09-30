package myvar_class

import (
	"sync/atomic"
	"sync"
	"math"

	"github.com/influxdata/influxdb/models"
)

// Var
type Var struct {
	measurement string
	tags        models.Tags
	name        string
	value       interface{}
}

func (v Var) GetMeasurement() string {
	return v.measurement
}

func (v Var) GetTags() models.Tags {
	return v.tags
}

func (v Var) GetValue() interface{} {
	return v.value
}

func (v Var) GetName() string {
	return v.name
}

// Int
type Int struct {
	key   string
	value int64
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

// Float
type Float struct {
	key   string
	value uint64
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

// String
type String struct {
	key   string
	value atomic.Value
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

// Map
type Map struct {
	key  string
	data sync.Map
}

func (m Map) GetData() sync.Map {
	return m.data
}

func (m Map) DelEntry(k interface{}) {
	m.data.Delete(k)
}

func (m Map) RangeFunc(f func(key, value interface{}) bool) {
	m.data.Range(f)
}

func (m *Map) Set(key string, value interface{}) *Map {
	m.data.Store(key, value)
	return m
}

func (m *Map) Get(key string) (interface{}, bool) {
	return m.data.Load(key)
}