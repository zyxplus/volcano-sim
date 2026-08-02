package proportion

import (
	"github.com/zhouyingxiao/volcano-sim/pkg/api"
	"testing"
)

func TestComputeDeservedAllocatesByWeight(t *testing.T) {
	queues := []api.Queue{
		{Name: "a", Weight: 1, Guarantee: api.NewResource(map[string]float64{"gpu": 8}), Capability: api.NewResource(map[string]float64{"gpu": 32}), Allocated: api.NewResource(map[string]float64{"gpu": 20})},
		{Name: "b", Weight: 3, Guarantee: api.NewResource(map[string]float64{"gpu": 8}), Capability: api.NewResource(map[string]float64{"gpu": 32})},
	}
	deserved := ComputeDeserved(queues, api.NewResource(map[string]float64{"gpu": 32}))
	if deserved["a"]["gpu"] != 8 || deserved["b"]["gpu"] != 24 {
		t.Fatalf("unexpected deserved: %#v", deserved)
	}
	if !IsOverused(queues[0], deserved["a"]) {
		t.Fatal("queue a should be overused")
	}
}
