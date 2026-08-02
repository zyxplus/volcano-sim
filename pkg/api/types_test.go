package api

import "testing"

func TestClusterTotalSumsNodeCapacity(t *testing.T) {
	nodes := []Node{
		{Name: "n1", Capacity: NewResource(map[string]float64{"gpu": 2})},
		{Name: "n2", Capacity: NewResource(map[string]float64{"gpu": 3, "cpu": 4})},
	}

	got := ClusterTotal(nodes)
	if got["gpu"] != 5 || got["cpu"] != 4 {
		t.Fatalf("unexpected total: %#v", got)
	}
}
