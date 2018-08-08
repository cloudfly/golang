package judge

import (
	"context"
	"fmt"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/client"
)

func TestLock(t *testing.T) {
	client, err := etcd.New(etcd.Config{Endpoints: []string{"http://localhost:2379"}})
	if err != nil {
		t.Fatal(err)
	}
	kv := etcd.NewKeysAPI(client)

	locker0 := NewLock(kv, "testlock", "testlockvalue")
	_, err = locker0.Lock(context.Background())
	if err != nil {
		t.Error(err)
	}

	go func() {
		locker1 := NewLock(kv, "testlock", "testlockvalue")
		_, err = locker1.Lock(context.Background())
		if err != nil {
			t.Error(err)
		}
		t.Log("locker1 get the lock")
	}()
	if err := locker0.Unlock(); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Millisecond * 500)
}

func TestRegister(t *testing.T) {
	config := &Config{
		Endpoints: []string{"http://localhost:2379"},
		Prefix:    "judge/v1",
		Advertise: "test1",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := Register(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	members, err := c.Members()
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Error("count of member not correct, should having one member")
	}
	if members[0] != "test1" {
		t.Errorf("member's name is not correct, %s != test1", members[0])
	}
	if c.Role() != Follower {
		t.Error("the role should be follower")
	}
}

func TestLeader(t *testing.T) {
	config := &Config{
		Endpoints: []string{"http://localhost:2379"},
		Prefix:    "judge/v1",
		Advertise: "test1",
		Election:  true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := Register(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	leader, err := c.Leader()
	if err != nil {
		t.Error(err)
	}
	if leader != "test1" {
		t.Errorf("the leader should be test1, but %s", leader)
	}
}
func TestWatchMembers(t *testing.T) {
	config := &Config{
		Endpoints: []string{"http://localhost:2379"},
		Prefix:    "judge/v1",
		Advertise: "test1",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := Register(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	members, err := c.WatchMembers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for member := range members {
			fmt.Println(member)
		}
	}()

	time.Sleep(time.Second)
	config = &Config{
		Endpoints: []string{"http://localhost:2379"},
		Prefix:    "judge/v1",
		Advertise: "test2",
	}
	c2, err := Register(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	mems, err := c2.Members()
	if err != nil {
		t.Error(err)
	}
	if len(mems) != 2 {
		t.Fatal("should having 2 members, but only one")
	}
	if mems[0] == "test1" || mems[1] == "test2" {
		return
	} else if mems[1] == "test1" || mems[0] == "test2" {
		return
	}
	t.Error("member 0 is not test1, 1 is not test2")
}

func TestWatchLeader(t *testing.T) {
	config := &Config{
		Endpoints: []string{"http://localhost:2379"},
		Prefix:    "judge/v1",
		Advertise: "test1",
		Election:  true,
	}
	ctx, cancel1 := context.WithCancel(context.Background())
	c, err := Register(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	config = &Config{
		Endpoints: []string{"http://localhost:2379"},
		Prefix:    "judge/v1",
		Advertise: "test2",
		Election:  true,
	}

	ctx, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	c2, err := Register(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	l, err := c2.Leader()
	if err != nil {
		t.Error(err)
	}
	if l != "test1" {
		t.Error("leader should be test1")
	}

	if c.Role() != Leader || c2.Role() != Follower {
		t.Error("the Role() method is uncorrect")
	}

	cancel1() // stop test1

	time.Sleep(time.Second * 1)

	// test2 should becoming the leader
	l, err = c2.Leader()
	if err != nil {
		t.Error(err)
	}
	if l != "test2" {
		t.Error("leader should be test2")
	}

	if c2.Role() != Leader {
		t.Error("the Role() method is uncorrect")
	}
}
