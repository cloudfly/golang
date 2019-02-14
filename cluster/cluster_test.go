package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCluster(t *testing.T) {

	testNodes := Nodes{
		{
			Name: "node1",
		},
		{
			Name: "node2",
		},
	}

	node1, err := New(context.Background(), []string{"localhost:2379"}, "/cluster-test", testNodes)
	assert.NoError(t, err)
	node2, err := New(context.Background(), []string{"localhost:2379"}, "/cluster-test", testNodes)
	assert.NoError(t, err)
	defer node1.Destroy()
	defer node2.Destroy()

	assert.Equal(t, testNodes, node2.Nodes())
	assert.Equal(t, testNodes, node1.Nodes())

	assert.NoError(t, node2.AddNode(Node{"node0", 3}))
	assert.NoError(t, node2.RemoveNode("node1"))
	assert.NoError(t, node2.UpdateNode(Node{"node2", 1}))

	time.Sleep(time.Second)
	nowNodes := node1.Nodes()
	assert.Equal(t, 2, len(nowNodes))

	assert.Equal(t, "node0", nowNodes[0].Name)
	assert.Equal(t, "node2", nowNodes[1].Name)
	assert.Equal(t, 3, nowNodes[0].Weight)
	assert.Equal(t, 1, nowNodes[1].Weight)
}

func TestCluster_NodesChan(t *testing.T) {
	testNodes := Nodes{
		{
			Name: "node1",
		},
		{
			Name: "node2",
		},
	}

	node1, err := New(context.Background(), []string{"localhost:2379"}, "/cluster-test", testNodes)
	assert.NoError(t, err)
	node2, err := New(context.Background(), []string{"localhost:2379"}, "/cluster-test", testNodes)
	assert.NoError(t, err)
	defer node1.Destroy()
	defer node2.Destroy()

	ch := node2.NodesChan()

	assert.NoError(t, node2.AddNode(Node{"node0", 3}))
	assert.NoError(t, node2.RemoveNode("node1"))
	assert.NoError(t, node2.UpdateNode(Node{"node2", 1}))

	nodes := <-ch
	assert.Equal(t, 3, len(nodes))
	assert.Equal(t, "node0", nodes[0].Name)
	assert.Equal(t, "node1", nodes[1].Name)
	assert.Equal(t, "node2", nodes[2].Name)

	nodes = <-ch
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, "node0", nodes[0].Name)
	assert.Equal(t, "node2", nodes[1].Name)

	nodes = <-ch
	assert.Equal(t, 1, nodes[1].Weight)
}
