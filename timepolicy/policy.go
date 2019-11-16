package timepolicy

import (
	"bytes"
	"context"
	"fmt"
	"time"
)

const (
	policySplit     = ','
	policyItemSplit = ':'
)

// Policy represent a group of policy item
type Policy struct {
	items []policyItem
	job   Job
	next  *Policy
}

// ParsePolicy 解析策略字符串
func ParsePolicy(from time.Time, s string) (Policy, error) {
	fields := bytes.Split([]byte(s), []byte{policySplit})
	policy := Policy{
		items: make([]policyItem, 0, len(fields)),
	}
	for _, field := range fields {
		item, err := parsePolicyItem(from, field)
		if err != nil {
			return Policy{}, err
		}
		policy.items = append(policy.items, item)
	}
	return policy, nil
}

// NextTime 返回下一次执行策略的时间, 基于参数 now 计算
func (policy *Policy) NextTime(now int64) int64 {
	var latest int64
	for _, item := range policy.items {
		next := item.next(now)
		if next != 0 && (next < latest || latest == 0) {
			latest = next
		}
	}
	return latest
}

type policyItem struct {
	Start    int64 // unix seconds
	Interval int64 // seconds
	End      int64 // unix seconds
}

func parsePolicyItem(from time.Time, s []byte) (policyItem, error) {
	fields := bytes.Split(s, []byte{policyItemSplit})
	durations := make([]time.Duration, 0, len(fields))
	for _, field := range fields {
		dur, err := time.ParseDuration(string(bytes.TrimSpace(field)))
		if err != nil {
			return policyItem{}, fmt.Errorf("uncorrect interval setting '%s': %s", field, err.Error())
		}
		durations = append(durations, dur)
	}

	item := policyItem{}
	switch len(fields) {
	case 1:
		item.Interval = int64(durations[0] / time.Second)
	case 2:
		item.Start = from.Add(durations[0]).Unix()
		item.Interval = int64(durations[1] / time.Second)
	case 3:
		item.Start = from.Add(durations[0]).Unix()
		item.Interval = int64(durations[1] / time.Second)
		item.End = from.Add(durations[2]).Unix()
	default:
		return policyItem{}, fmt.Errorf("uncorrect policy '%s'", s)
	}
	if item.Start != 0 && item.End != 0 && item.Start > item.End {
		return policyItem{}, fmt.Errorf("the start time after end time")
	}
	if item.Interval <= 0 {
		return policyItem{}, fmt.Errorf("uncorrect interval setting, should longger(or equal) than 1s")
	}
	return item, nil
}

func (item policyItem) next(now int64) int64 {
	if item.Start != 0 && now < item.Start {
		return item.Start
	}
	if item.End != 0 && now > item.End {
		return 0
	}
	var mod int64
	if item.Start == 0 {
		mod = now % item.Interval
	} else {
		mod = (now - item.Start) % item.Interval
	}
	if mod == 0 {
		return now
	}
	next := now + item.Interval - mod
	if item.End > 0 && next > item.End {
		return 0
	}
	return next
}

// Job represents the job type which the engine will call it by go Func()
type Job interface {
	Do(time.Time)
	Finished() bool
}

// Engine will run process the policy and run the function at a right time
type Engine struct {
	ctx   context.Context
	ch    chan engineCommand
	table map[int64]*Policy
}

// NewEngine create a engine
func NewEngine(ctx context.Context) *Engine {
	engine := Engine{
		ctx: ctx,
		ch:  make(chan engineCommand, 10),
	}
	return &engine
}

// RegisterWithTime a policy to engine
func (engine *Engine) RegisterWithTime(from time.Time, policy string, job Job) error {
	if job == nil {
		return nil
	}
	p, err := ParsePolicy(from, policy)
	if err != nil {
		return err
	}
	p.job = job
	engine.ch <- scheCommand{&p, time.Now()}
	return nil
}

// Register a policy to engine with using time.Now() as from time
func (engine *Engine) Register(policy string, job Job) error {
	return engine.RegisterWithTime(time.Now(), policy, job)
}

func (engine *Engine) activate() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	done := engine.ctx.Done()

	for {
		select {
		case cmd := <-engine.ch:
			cmd.execute(engine)
		case now := <-ticker.C:
			unix := now.Unix()
			policy, ok := engine.table[unix]
			if !ok {
				break
			}
			go func(p *Policy, t time.Time) {
				iter := p
				for iter != nil {
					if !iter.job.Finished() {
						go iter.job.Do(t)
						engine.ch <- scheCommand{iter, t.Add(time.Second)}
					}
					iter = iter.next
				}
			}(policy, now)
			delete(engine.table, unix)
		case <-done:
			return
		}
	}
}

type engineCommand interface {
	execute(*Engine)
}

type scheCommand struct {
	p    *Policy
	from time.Time
}

func (cmd scheCommand) execute(engine *Engine) {
	nextUnix := cmd.p.NextTime(cmd.from.Unix())
	if nextUnix > 0 {
		cmd.p.next = engine.table[nextUnix]
		engine.table[nextUnix] = cmd.p
	}
}

type clearCommand struct{}

func (cmd clearCommand) execute(engine *Engine) {
	engine.table = make(map[int64]*Policy)
}
