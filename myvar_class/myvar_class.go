package myvar_class

import (
	"fmt"
	"sync"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	myv "github.com/cloudfly/golang/myvar"
	"github.com/influxdata/influxdb/models"
	log "github.com/Sirupsen/logrus"
)

type InstanceMyVar struct {
	flushInterval time.Duration
	cacheLock     sync.Mutex
	cache         map[string]*myv.Var
	tempCache     []*client.Point
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

	_, _, err = cli.Ping(time.Second * 3)
	if err != nil && len(err.Error()) != 0 {
		return nil, fmt.Errorf("influxdb connection failed, %s", err.Error())
	}

	mv := &InstanceMyVar{
		c: cli,
		database: db,
		flushInterval: interval,
		cache: make(map[string]*myv.Var),
		tempCache: make([]*client.Point, 0, 1000),
	}

	close(mv.cancel)
	mv.cancel = make(chan struct{})
	go mv.flusher()

	return mv, nil
}

// SetGlobalTag set global tag list, used by each variable
func (mv *InstanceMyVar) SetGlobalTag(key, value string) {
	mv.gtags.SetString(key, value)
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
		case *myv.Float:
			fields[v.GetName()] = vv.Value()
		case *myv.Int:
			fields[v.GetName()] = vv.Value()
		case *myv.String:
			s := vv.Value()
			if len(s) == 0 {
				continue LOOP
			}
			fields[v.GetName()] = s
			vv.Set("") // 重置一下, 下次除非有新的数据过来, 否则就不发了
		case *myv.Map:
			vv.RangeFunc(func(k, v interface{}) bool {
				fields[k.(string)] = v
				vv.DelEntry(k)
				return true
			})
		}
		if len(fields) == 0 {
			continue
		}
		p, err := models.NewPoint(v.GetMeasurement(), append(v.GetTags().Clone(), mv.gtags...), models.Fields(fields), t)
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