package influxdb_client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
	influxmodels "github.com/influxdata/influxdb/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	bufPool = new(sync.Pool)
)

// InfluxdbClient station
type InfluxdbClient struct {
	locker         *sync.RWMutex
	wg             *sync.WaitGroup
	ctx            context.Context
	addr           string
	maxCachePoints int
	timeout        time.Duration
	client         influxdb.Client
	blockOnError   bool
	processors     map[string]*processor
}

func init() {
	bufPool.New = func() interface{} {
		return make([]*influxdb.Point, 0, 5000)
	}
}

// New InfluxDB component
func NewInfluxdbClient(ctx context.Context, options ...ClientOption) (*InfluxdbClient, error) {
	client := &InfluxdbClient{
		ctx:            ctx,
		locker:         &sync.RWMutex{},
		wg:             &sync.WaitGroup{},
		addr:           "http://127.0.0.1:8086",
		maxCachePoints: 1000,
		timeout:        time.Second * 30,
		processors:     make(map[string]*processor),
	}

	for _, op := range options {
		op(client)
	}

	// create influxdb client
	c, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:               client.addr,
		Timeout:            client.timeout, // 30 seconds timeout
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create influxdb http client")
	}
	client.client = c

	return client, nil
}

// Request execute a query on influxdb, and return the raw response
func (client *InfluxdbClient) Request(sql, database, precision string) (*influxdb.Response, error) {
	resp, err := client.client.Query(influxdb.NewQuery(sql, database, precision))
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	return resp, nil
}

// Query execute the given sql query on influxdb
func (client *InfluxdbClient) Query(sql, database, precision string) ([]influxmodels.Row, error) {
	resp, err := client.Request(sql, database, precision)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 {
		return []influxmodels.Row{}, nil
	}
	result := resp.Results[0]
	if result.Err != "" {
		return nil, errors.New(result.Err)
	}
	return result.Series, nil
}

// Databases return the database list on influxdb
func (client *InfluxdbClient) Databases() ([]string, error) {
	sql := "SHOW DATABASES"
	rows, err := client.Query(sql, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "Fail to execute SHOW DATABASES")
	}
	databases := make([]string, 0, len(rows))
	for _, row := range rows {
		for _, value := range row.Values {
			databases = append(databases, fmt.Sprintf("%s", value[0]))
		}
	}
	return databases, nil
}

// Measurements return the measurement list on a specify database
func (client *InfluxdbClient) Measurements(database string) ([]string, error) {
	sql := "SHOW MEASUREMENTS"
	rows, err := client.Query(sql, database, "")
	if err != nil {
		return nil, errors.Wrap(err, "Fail to execute SHOW MEASUREMENTS")
	}
	measurements := make([]string, 0, len(rows))
	for _, row := range rows {
		for _, value := range row.Values {
			measurements = append(measurements, fmt.Sprintf("%s", value[0]))
		}
	}
	return measurements, nil
}

// Tags get all tag keys from influxdb for group-artifact
func (client *InfluxdbClient) Tags(database string, measurements []string) (map[string][]string, error) {
	var queries []string
	for _, measurement := range measurements {
		queries = append(queries, fmt.Sprintf("SHOW TAG KEYS FROM \"%s\"", measurement))
	}
	resp, err := client.Request(strings.Join(queries, ";"), database, "")
	if err != nil {
		return nil, err
	}

	data := make(map[string][]string)
	for _, result := range resp.Results {
		for _, row := range result.Series {
			if _, ok := data[row.Name]; !ok {
				data[row.Name] = make([]string, 0, 1)
			}
			for _, value := range row.Values {
				data[row.Name] = append(data[row.Name], fmt.Sprintf("%s", value[0]))
			}
		}
	}
	return data, nil
}

// FieldKey represents a result's structure from influxdb with query `SHOW FIELD KEYS FROM ...`
type FieldKey struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Fields get all tag keys from influxdb in given measurements
func (client *InfluxdbClient) Fields(database string, measurements []string) (map[string][]FieldKey, error) {
	var queries []string
	for _, measurement := range measurements {
		queries = append(queries, fmt.Sprintf("SHOW FIELD KEYS FROM \"%s\"", measurement))
	}
	resp, err := client.Request(strings.Join(queries, ";"), database, "")
	if err != nil {
		return nil, err
	}

	data := make(map[string][]FieldKey)
	for _, result := range resp.Results {
		for _, row := range result.Series {
			if _, ok := data[row.Name]; !ok {
				data[row.Name] = make([]FieldKey, 0, 1)
			}
			for _, value := range row.Values {
				data[row.Name] = append(data[row.Name], FieldKey{
					Name: fmt.Sprintf("%s", value[0]),
					Type: fmt.Sprintf("%s", value[1]),
				})
			}
		}
	}
	return data, nil
}

// Write influxdb point data
func (client *InfluxdbClient) Write(database, rp string, points []*influxdb.Point) error {
	for _, item := range points {
		if item == nil {
			continue
		}
		processor, err := client.getProcesser(database, rp)
		if err != nil {
			return err
		}
		processor.writeCache(points)
	}
	return nil
}

// WaitExit block until all the processer exit
func (client *InfluxdbClient) WaitExit() {
	<-client.ctx.Done()
	client.wg.Wait()
}

func (client *InfluxdbClient) getProcesser(name, rp string) (*processor, error) {
	if name == "" {
		return nil, errors.New("database name is empty")
	}
	key := fmt.Sprintf("%s %s", name, rp)
	client.locker.RLock()
	processor, ok := client.processors[key]
	client.locker.RUnlock()
	if !ok {
		var err error
		processor, err = newProcessor(client.ctx, client.wg, client.addr, name, rp, client.maxCachePoints, client.timeout, client.blockOnError)
		if err != nil {
			return nil, err
		}
		client.locker.Lock()
		client.processors[key] = processor
		client.locker.Unlock()
	}
	return processor, nil
}

// database represents a influxdb client of specific database
type processor struct {
	*sync.WaitGroup
	cacheLock sync.Mutex
	flushLock sync.Mutex
	ctx       context.Context

	database      string
	rp            string
	client        influxdb.Client
	cache         []*influxdb.Point
	maxPointCache int
	blockOnError  bool

	logger *log.Entry
}

// newProcessor create a new Influx, it return error if fail to connect to influxdb
func newProcessor(ctx context.Context, wg *sync.WaitGroup, addr, database, rp string, maxPointCache int, timeout time.Duration, blockOnError bool) (*processor, error) {
	config := influxdb.HTTPConfig{
		Addr:               addr,
		Timeout:            timeout, // 30 seconds timeout
		InsecureSkipVerify: true,
	}
	// create influxdb client
	client, err := influxdb.NewHTTPClient(config)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create influxdb http client")
	}

	p := &processor{
		ctx:           ctx,
		WaitGroup:     wg,
		database:      database,
		rp:            rp,
		maxPointCache: maxPointCache,
		cache:         bufPool.Get().([]*influxdb.Point),
		client:        client,
		logger:        log.WithField("database", database),
		blockOnError:  blockOnError,
	}

	go p.flusher()

	return p, nil
}

func (p *processor) writeCache(points []*influxdb.Point) {
	p.cacheLock.Lock()
	p.cache = append(p.cache, points...)
	// 超过 maxPointCache 马上 flush 一次, 最好不要超过 5000 个 points 再 flush(官方推荐)
	if len(p.cache) > p.maxPointCache {
		data := p.cache
		p.cache = bufPool.Get().([]*influxdb.Point)
		p.cacheLock.Unlock()
		p.flush(data)
	} else {
		p.cacheLock.Unlock()
	}
}

// flusher will flush the points into influxdb
func (p *processor) flusher() {
	p.Add(1)
	defer p.Done()
	done := p.ctx.Done()

	n := 0
	// resetTick 用来定期检查 qps, 进而调整 flush 频率
	resetTick := time.Tick(time.Second * 30)
	// 默认刷新间隔 30 毫秒
	interval := 30
	tick := time.Tick(time.Millisecond * time.Duration(interval))
	for {
		select {
		case <-resetTick:
			// 计算 QPS
			qps := n / 30
			n = 0
			// QPS <= 100: flush 频率设成 500ms, 绝大多数是低于 100 的
			// 100 < QPS <= 1000: flush 频率设成 200ms, 存在一小部分
			// QPS > 1000: flush 频率设成 30ms, 一般就三两个库会有这种流量
			// 如果 qps 在 100 上下来回波动, 会导致 tick 被不停的被重置,
			// 但毕竟 30s 才重置一次, 对性能影响不会很大
			if qps <= 100 && interval != 500 {
				interval = 500
			} else if qps > 100 && qps <= 1000 && interval != 200 {
				interval = 200
			} else if qps > 1000 && interval != 30 {
				interval = 30
			} else {
				// 没有改变, 直接 break the select
				break
			}
			// 重置 ticker, 就的 ticker 会被 gc 掉的, 不用管
			tick = time.Tick(time.Millisecond * time.Duration(interval))
		case <-done:
			p.flush(p.cache)
			return
		case <-tick:
			if len(p.cache) == 0 {
				break // break the select
			}
			p.cacheLock.Lock()
			data := p.cache
			p.cache = bufPool.Get().([]*influxdb.Point)
			p.cacheLock.Unlock()
			n += len(data)
			p.flush(data)
		}
	}
}

func (p *processor) flush(data []*influxdb.Point) {
	defer bufPool.Put(data[0:0])
	p.flushLock.Lock()
	defer p.flushLock.Unlock()
	batch, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:        p.database,
		RetentionPolicy: p.rp,
	})
	if err != nil {
		p.logger.Errorf("fail to create point batch, %s", err.Error())
		return
	}
	batch.AddPoints(data)

	for {
		// write into influxdb
		if err := p.client.Write(batch); err != nil {
			errMsg := err.Error()
			if len(errMsg) > 0 && errMsg[0] == '{' {
				// the influxdb will return a json format error for request error
				// so the error message must start with '{'
				// write the failed point to exception measurement
			} else if p.blockOnError {
				p.logger.Warn("retry to flush points into influxdb every seconds until influxdb recovery")
				time.Sleep(time.Second)
				continue
			}
		}
		break
	}
}

type ClientOption func(client *InfluxdbClient)

func WithMaxCachePoint(max int) ClientOption {
	return func(client *InfluxdbClient) {
		client.maxCachePoints = max
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(client *InfluxdbClient) {
		client.timeout = timeout
	}
}

func WithBlockOnError() ClientOption {
	return func(client *InfluxdbClient) {
		client.blockOnError = true
	}
}

func WithAddr(addr string) ClientOption {
	return func(client *InfluxdbClient) {
		client.addr = addr
	}
}
