package myvar_class

import (
	"fmt"
	"sync"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/influxdata/influxdb/models"
	log "github.com/Sirupsen/logrus"
	"sort"
)

type InstanceMyVar struct {
	flushInterval time.Duration
	cacheLock     sync.Mutex
	cache         map[string]*Var // cache of Var instances
	tempCache     []*client.Point // cache of influxdb points
	cancel        chan struct{}
	gtags         models.Tags

	database string
	c        client.Client
}

// NewMyVar inits an instance of MyVarClass
func NewMyVar(addr, db string, interval time.Duration) (*InstanceMyVar, error) {

	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:    addr,
		Timeout: time.Second * 10,
	})
	if err != nil {
		return nil, err
	}

	_, _, err = cli.Ping(time.Second * 3)
	if err != nil && len(err.Error()) != 0 {
		return nil, fmt.Errorf("influxdb connection failed, %s", err.Error())
	}

	// create db
	cli.Query(client.NewQuery(fmt.Sprintf(`create database "%s"`, db), "", ""))

	mv := &InstanceMyVar{
		c: cli,
		database: db,
		flushInterval: interval,
		cancel: make(chan struct{}),
		cache: make(map[string]*Var),
		tempCache: make([]*client.Point, 0, 1000),
	}
	go mv.flusher()

	return mv, nil
}

// SetGlobalTag set global tag list, used by each variable
func (mv *InstanceMyVar) SetGlobalTag(key, value string) {
	mv.gtags.SetString(key, value)
}

// GetDatabase returns db used for monitoring
func (mv *InstanceMyVar) GetDatabase() string {
	return mv.database
}

func (mv *InstanceMyVar) flusher() {
	ticker := time.Tick(mv.flushInterval)
	for {
		select {
		case t := <-ticker:
			if mv.database == "" || mv.c == nil {
				break
			}
			if err := mv.Flush(t.Truncate(time.Second)); err != nil {
				log.Errorf("failed to flush points into influxdb, %s", err.Error())
			}
		case <-mv.cancel:
			if err := mv.Flush(time.Now().Truncate(time.Second)); err != nil {
				log.Errorf("failed to flush points into influxdb, %s", err.Error())
			}
			return
		}
	}
}

// Flush write all the points in cache into influxdb
func (mv *InstanceMyVar) Flush(tt ...time.Time) error {
	var (
		t time.Time
	)
	if len(tt) > 0 {
		t = tt[0]
	} else {
		t = time.Now()
	}

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: mv.database,
	})
	if err != nil {
		log.Errorf("fail to create influxdb batch, %s", err.Error())
		return err
	}

	mv.cacheLock.Lock()
LOOP:
	for _, v := range mv.cache {
		fields := map[string]interface{}{}
		switch vv := v.GetValue().(type) {
		case *Float:
			fields[v.GetName()] = vv.Value()
		case *Int:
			fields[v.GetName()] = vv.Value()
		case *String:
			s := vv.Value()
			if len(s) == 0 {
				continue LOOP
			}
			fields[v.GetName()] = s
			vv.Set("") // 重置一下, 下次除非有新的数据过来, 否则就不发了
		case *Map:
			vv.RangeFunc(func(k, v interface{}) bool {
				fields[k.(string)] = v
				vv.DelEntry(k)
				return true
			})
		}
		if len(fields) == 0 {
			continue
		}
		tagList := append(v.GetTags().Clone(), mv.gtags...)
		sort.Sort(tagList)
		p, err := models.NewPoint(v.GetMeasurement(), tagList, models.Fields(fields), t)
		if err != nil {
			log.Errorf("failed create new influxdb point, %s", err.Error())
			continue
		}
		batch.AddPoint(client.NewPointFrom(p))
	}

	if len(mv.tempCache) > 0 {
		batch.AddPoints(mv.tempCache)
		mv.tempCache = mv.tempCache[:0]
	}
	mv.cacheLock.Unlock()

	if len(batch.Points()) == 0 {
		return nil
	}
	return mv.c.Write(batch)
}

// Publish a raw influxdb points
func (mv *InstanceMyVar) Publish(name string, tags map[string]string, fields map[string]interface{}) error {
	p, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		return err
	}
	mv.cacheLock.Lock()
	defer mv.cacheLock.Unlock()
	mv.tempCache = append(mv.tempCache, p)
	return nil
}

func (mv *InstanceMyVar) publish(measurement string, tags map[string]string, name string, value interface{}) {
	mv.cacheLock.Lock()
	defer mv.cacheLock.Unlock()

	mv.cache[key(measurement, tags, name)] = &Var{
		measurement: measurement,
		tags:        models.NewTags(tags),
		name:        name,
		value:       value,
	}
}

func (mv *InstanceMyVar) getVar(k string) (*Var, bool) {
	mv.cacheLock.Lock()
	defer mv.cacheLock.Unlock()

	v, ok := mv.cache[k]
	return v, ok
}

//
func (mv *InstanceMyVar) NewInt(measurement string, tags map[string]string, name string) *Int {
	k := key(measurement, tags, name)

	data, ok := mv.getVar(k)
	if ok {
		n, ok := data.GetValue().(*Int)
		if ok {
			return n
		}
	}
	n := new(Int)
	n.key = k
	mv.publish(measurement, tags, name, n)
	return n
}

func (mv *InstanceMyVar) FreeInt(n *Int) {
	mv.cacheLock.Lock()
	delete(mv.cache, n.key)
	mv.cacheLock.Unlock()
}

//
func (mv *InstanceMyVar) NewFloat(measurement string, tags map[string]string, name string) *Float {
	k := key(measurement, tags, name)

	data, ok := mv.getVar(k)
	if ok {
		n, ok := data.GetValue().(*Float)
		if ok {
			return n
		}
	}
	n := new(Float)
	n.key = k
	mv.publish(measurement, tags, name, n)
	return n
}

func (mv *InstanceMyVar) FreeFloat(f *Float) {
	mv.cacheLock.Lock()
	delete(mv.cache, f.key)
	mv.cacheLock.Unlock()
}

//
func (mv *InstanceMyVar) NewString(measurement string, tags map[string]string, name string) *String {
	k := key(measurement, tags, name)

	data, ok := mv.getVar(k)
	if ok {
		s, ok := data.value.(*String)
		if ok {
			return s
		}
	}
	s := new(String)
	s.key = k
	mv.publish(measurement, tags, name, s)
	return s
}

func (mv *InstanceMyVar) FreeString(s *String) {
	mv.cacheLock.Lock()
	delete(mv.cache, s.key)
	mv.cacheLock.Unlock()
}

//
func (mv *InstanceMyVar) NewMap(measurement string, tags map[string]string) *Map {
	k := key(measurement, tags, "")

	data, ok := mv.getVar(k)
	if ok {
		m, ok := data.value.(*Map)
		if ok {
			return m
		}
	}
	m := new(Map)
	m.key = k
	mv.publish(measurement, tags, "", m)
	return m
}

func (mv *InstanceMyVar) FreeMap(m *Map) {
	mv.cacheLock.Lock()
	delete(mv.cache, m.key)
	mv.cacheLock.Unlock()
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