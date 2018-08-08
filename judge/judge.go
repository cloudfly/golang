package judge

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cloudfly/log"
	etcd "github.com/coreos/etcd/client"
	"github.com/pkg/errors"
)

// keys' ttl
var (
	LockTTL          = time.Second * 10
	RegisterTTL      = time.Second * 10
	RegisterInterval = time.Second * 3
)

// Config for create cluster
type Config struct {
	Endpoints []string
	Prefix    string
	Advertise string
	Election  bool
}

// Register create a new Candidate on specific service, and return it. icandidate quit the service when context was canceled
func Register(ctx context.Context, config *Config) (*Candidate, error) {
	client, err := etcd.New(etcd.Config{
		Endpoints: config.Endpoints,
	})
	if err != nil {
		return nil, errors.Wrap(err, "fail to create etcd client")
	}
	c := &Candidate{
		kv:        etcd.NewKeysAPI(client),
		prefix:    config.Prefix,
		advertise: config.Advertise,
		ctx:       ctx,
		conf:      config,
	}
	if c.prefix[0] == '/' {
		c.prefix = c.prefix[1:]
	}
	_, err = c.kv.Set(ctx, fmt.Sprintf("%s/members/%s", c.prefix, c.advertise), c.advertise, &etcd.SetOptions{TTL: RegisterTTL})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to join cluster")
	}
	go c.hold()
	if config.Election {
		go c.elect()
	}
	return c, nil
}

// Candidate represent a member in Cluster
type Candidate struct {
	kv        etcd.KeysAPI
	prefix    string
	locker    *Lock
	advertise string
	err       error
	ctx       context.Context
	conf      *Config
}

func (c *Candidate) hold() {
	t := RegisterInterval
	if t == 0 {
		t = RegisterTTL / 5
	}
	ticker := time.Tick(t)
	key := fmt.Sprintf("%s/members/%s", c.prefix, c.advertise)
	logger := log.With("advertise", c.advertise).With("prefix", c.prefix)
	ctx := context.Background()
	done := c.ctx.Done()
	for {
		select {
		case <-ticker:
			if _, err := c.kv.Set(ctx, key, "", &etcd.SetOptions{TTL: RegisterTTL, Refresh: true}); err != nil {
				logger.Warn("fail to update %s on store, %s, try to recreate it", key, err.Error())
				if etcd.IsKeyNotFound(err) {
					c.kv.Set(ctx, key, c.advertise, &etcd.SetOptions{TTL: RegisterTTL})
				}
			}
		case <-done:
			// return directly, do not delete the member key but waiting ttl itself.
			// because sometimes candidate just want to restart, and register back again very soon.
			return
		}
	}
}

func (c *Candidate) elect() {
	logger := log.With("advertise", c.advertise).With("prefix", c.prefix)
	logger.Debug("join the service")

	var (
		lost <-chan struct{}
		err  error
	)

	// try to join service, retry 3 times in case of some error happend
START:
	for i := 0; i < 3; i++ {
		// Join will block until becoming leader
		c.locker = NewLock(c.kv, fmt.Sprintf("%s/leader", c.prefix), c.advertise)
		lost, err = c.locker.Lock(c.ctx)

		// three situations can reach here
		// 1. candidate get the leader successfully
		// 2. context was closed, quit the election
		// 3. some unexpected error happened

		if c.ctx.Err() != nil { // context was canceled
			logger.Debug("Election was canceled, quit.")
			return
		}
		if err != nil { // some unexpected error
			logger.Error("Fail to join the election, %s, retry in 3 seconds", err.Error())
			time.Sleep(time.Second * 3)
			goto START
		}
		// only leader can come here
		break
	}
	if err != nil { // still error after retry 3 times
		logger.Fatal("Fail to join election for 3 times, give up. %s", err.Error())
	}
	logger.Debug("Becomming a leader")
	select {
	case <-lost:
		logger.Warn("Lost the leader identity")
		goto START
	case <-c.ctx.Done():
		if err := c.locker.Unlock(); err != nil {
			if c.err == nil {
				c.err = err
			}
			logger.Error("fail to unlock leader")
		}
		logger.Debug("quit the election")
	}
}

// Leader return the leader address
func (c *Candidate) Leader() (string, error) {
	resp, err := c.kv.Get(context.Background(), fmt.Sprintf("%s/leader", c.prefix), nil)
	if err != nil {
		if err.(etcd.Error).Code == etcd.ErrorCodeKeyNotFound {
			time.Sleep(time.Millisecond * 200)
			return c.Leader()
		}
		return "", err
	}
	return resp.Node.Value, nil
}

// Members return all the members
func (c *Candidate) Members() ([]string, error) {
	key := fmt.Sprintf("%s/members", c.prefix)
	resp, err := c.kv.Get(context.Background(), key, &etcd.GetOptions{
		Recursive: true,
	})
	if err != nil {
		if err.(etcd.Error).Code == etcd.ErrorCodeKeyNotFound {
			time.Sleep(time.Millisecond * 200)
			return c.Members()
		}
		return nil, err
	}
	members := make([]string, len(resp.Node.Nodes))
	for i, node := range resp.Node.Nodes {
		members[i] = node.Value
	}
	return members, nil
}

// WatchLeader return a channel, which output leader value when leader changed
func (c *Candidate) WatchLeader(ctx context.Context) (<-chan string, error) {
	if !c.conf.Election {
		return nil, errors.New("candidate did not join a election")
	}

	leader, err := c.Leader()
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/leader", c.prefix)
	watcher := c.kv.Watcher(key, nil)
	leaderCh := make(chan string)

	go func() {
		defer close(leaderCh)
		leaderCh <- leader
		for {
			resp, err := watcher.Next(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// context was canceled
					break
				} else {
					// error unexpected, retry in 3 seconds
					time.Sleep(time.Second * 3)
					continue
				}
			}
			if resp.Node.Value != leader {
				leaderCh <- resp.Node.Value
				leader = resp.Node.Value
			}
		}
	}()
	return leaderCh, nil
}

// WatchMembers return a channel, which output the member list when updated
func (c *Candidate) WatchMembers(ctx context.Context) (<-chan []string, error) {
	members, err := c.Members()
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/members", c.prefix)
	watcher := c.kv.Watcher(key, &etcd.WatcherOptions{Recursive: true})
	memberCh := make(chan []string)

	go func() {
		defer close(memberCh)
		memberCh <- members
		for {
			resp, err := watcher.Next(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// context was canceled
					break
				} else {
					// error unexpected, retry in 3 seconds
					time.Sleep(time.Second * 3)
					continue
				}
			}
			if resp.Action == "delete" || resp.Action == "expire" || resp.Action == "compareAndDelete" {
				if resp.PrevNode != nil && resp.PrevNode.Value == c.advertise {
					// Actually, self can not be disappeared unless context was canceled
					// It must be the etcd's mistake that lose out member key.
					// So we ignore this case
					continue
				}
			}
			latest, err := c.Members()
			if err != nil {
				continue
			}
			if !sameArray(members, latest) {
				members = latest
				memberCh <- members
			}

		}
	}()
	return memberCh, nil
}

func sameArray(arr1, arr2 []string) bool {
	if arr1 == nil && arr2 == nil {
		return true
	}
	if arr1 == nil || arr2 == nil {
		return false
	}
	if len(arr1) != len(arr2) {
		return false
	}
	sort.Strings(arr1)
	sort.Strings(arr2)
	for i, item1 := range arr1 {
		if item1 != arr2[i] {
			return false
		}
	}
	return true
}

// Store return a KeysAPI of etcd
func Store(endpoints []string) (etcd.KeysAPI, error) {
	addrs := make([]string, len(endpoints))
	for i, endpoint := range endpoints {
		addrs[i] = fmt.Sprintf("http://%s", endpoint)
	}
	client, err := etcd.New(etcd.Config{
		Endpoints: addrs,
	})
	if err != nil {
		return nil, errors.Wrap(err, "fail to create etcd client")
	}
	return etcd.NewKeysAPI(client), nil
}

// Lock represent a distribute lock
type Lock struct {
	client    etcd.KeysAPI
	stopLock  chan struct{}
	stopRenew chan struct{}
	key       string
	value     string
	last      *etcd.Response
}

// NewLock returns a handle to a lock struct which can
// be used to provide mutual exclusion on a key
func NewLock(kv etcd.KeysAPI, key, value string) *Lock {
	// Create lock object
	return &Lock{
		client:    kv,
		stopRenew: make(chan struct{}),
		key:       key,
		value:     value,
	}
}

// Lock attempts to acquire the lock and blocks while
// doing so. It returns a channel that is closed if our
// lock is lost or if an error occurs
func (l *Lock) Lock(ctx context.Context) (<-chan struct{}, error) {
	// Lock holder channel
	lockHeld := make(chan struct{})

	setOpts := &etcd.SetOptions{
		TTL: LockTTL,
	}

	for {
		setOpts.PrevExist = etcd.PrevNoExist
		resp, err := l.client.Set(context.Background(), l.key, l.value, setOpts)
		if err != nil {
			if etcdError, ok := err.(etcd.Error); ok {
				if etcdError.Code != etcd.ErrorCodeNodeExist {
					return nil, err
				}
				setOpts.PrevIndex = ^uint64(0)
			}
		} else {
			setOpts.PrevIndex = resp.Node.ModifiedIndex
		}

		setOpts.PrevExist = etcd.PrevExist
		l.last, err = l.client.Set(context.Background(), l.key, l.value, setOpts)

		if err == nil {
			// Leader section
			l.stopLock = make(chan struct{})
			go l.holdLock(l.key, lockHeld)
			break
		} else {
			// If this is a legitimate error, return
			if etcdError, ok := err.(etcd.Error); ok {
				if etcdError.Code == etcd.ErrorCodeKeyNotFound {
					// key was deleted
					continue
				} else if etcdError.Code != etcd.ErrorCodeTestFailed {
					return nil, err
				}
			}
			// Seeker section
			errorCh := make(chan error)
			chWStop := make(chan bool)
			free := make(chan bool)

			go l.waitLock(l.key, errorCh, chWStop, free)

			// Wait for the key to be available or for
			// a signal to stop trying to lock the key
			select {
			case <-free:
				break
			case err := <-errorCh:
				return nil, err
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			// Delete or Expire event occurred
			// Retry
		}
	}

	return lockHeld, nil
}

// Hold the lock as long as we can
// Updates the key ttl periodically until we receive
// an explicit stop signal from the Unlock method
func (l *Lock) holdLock(key string, lockHeld chan struct{}) {
	defer close(lockHeld)

	ticker := time.Tick(LockTTL / 5)

	var err error
	setOpts := &etcd.SetOptions{TTL: LockTTL, Refresh: true}

	for {
		select {
		case <-ticker:
			setOpts.PrevIndex = l.last.Node.ModifiedIndex
			l.last, err = l.client.Set(context.Background(), key, "", setOpts)
			if err != nil {
				return
			}

		case <-l.stopLock:
			return
		}
	}
}

// WaitLock simply waits for the key to be available for creation
func (l *Lock) waitLock(key string, errorCh chan error, stopWatchCh chan bool, free chan<- bool) {
	opts := &etcd.WatcherOptions{Recursive: false}
	watcher := l.client.Watcher(key, opts)

	for {
		event, err := watcher.Next(context.Background())
		if err != nil {
			errorCh <- err
			return
		}
		if event.Action == "delete" || event.Action == "expire" || event.Action == "compareAndDelete" {
			free <- true
			return
		}
	}
}

// Unlock the "key". Calling unlock while
// not holding the lock will throw an error
func (l *Lock) Unlock() error {
	if l.stopLock != nil {
		l.stopLock <- struct{}{}
	}
	if l.last != nil {
		delOpts := &etcd.DeleteOptions{
			PrevIndex: l.last.Node.ModifiedIndex,
		}
		_, err := l.client.Delete(context.Background(), l.key, delOpts)
		if err != nil {
			return err
		}
	}
	return nil
}
