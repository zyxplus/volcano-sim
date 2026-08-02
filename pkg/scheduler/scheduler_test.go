package scheduler

import (
	"testing"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
)

func TestOrderJobsUsesLowestDominantShareFirst(t *testing.T) {
	scheduler := New([]api.Node{{
		Name:     "n",
		Capacity: api.NewResource(map[string]float64{"cpu": 100, "gpu": 10}),
	}})
	jobs := []api.Job{
		{Name: "gpu-heavy", Allocated: api.NewResource(map[string]float64{"gpu": 5})},
		{Name: "cpu-heavy", Allocated: api.NewResource(map[string]float64{"cpu": 20})},
	}

	got := scheduler.OrderJobs(jobs)
	if got[0].Name != "cpu-heavy" {
		t.Fatalf("got %q first, want cpu-heavy", got[0].Name)
	}
}

func TestRunRollsBackJobWhenGangCannotReachMinimum(t *testing.T) {
	scheduler := New([]api.Node{{
		Name:     "n1",
		Capacity: api.NewResource(map[string]float64{"gpu": 2}),
		Idle:     api.NewResource(map[string]float64{"gpu": 2}),
	}})

	plan := scheduler.Run([]api.Job{{
		Name:         "four-gpu-tasks",
		MinAvailable: 4,
		Replicas:     4,
		Request:      api.NewResource(map[string]float64{"gpu": 1}),
	}})

	if len(plan.Allocations) != 0 {
		t.Fatalf("partial gang allocation leaked: %#v", plan.Allocations)
	}
	if plan.Unschedulable["four-gpu-tasks"] != "insufficient resources for minAvailable" {
		t.Fatalf("missing unschedulable reason: %#v", plan.Unschedulable)
	}
}

func TestRunRejectsRequiredGangSplitAcrossFabrics(t *testing.T) {
	nodes := []api.Node{
		{
			Name:     "fabric-a-node",
			Capacity: api.NewResource(map[string]float64{"gpu": 16}),
			Labels:   map[string]string{"fabric-id": "fabric-a", "gpu-model": "c550"},
		},
		{
			Name:     "fabric-b-node",
			Capacity: api.NewResource(map[string]float64{"gpu": 16}),
			Labels:   map[string]string{"fabric-id": "fabric-b", "gpu-model": "c550"},
		},
	}
	job := api.Job{
		Name:         "large",
		MinAvailable: 24,
		Replicas:     24,
		Request:      api.NewResource(map[string]float64{"gpu": 1}),
		Topology:     &api.Topology{GPUModel: "c550", SameFabric: "Required"},
	}

	plan := New(nodes).Run([]api.Job{job})
	if len(plan.Allocations) != 0 {
		t.Fatalf("split Fabric allocation leaked: %#v", plan.Allocations)
	}
	if plan.Unschedulable["large"] != "no single fabric has enough capacity" {
		t.Fatalf("unexpected unschedulable reason: %#v", plan.Unschedulable)
	}
}

func TestRunPlacesRequiredGangInOneFeasibleFabric(t *testing.T) {
	nodes := []api.Node{
		{
			Name:     "fabric-a-node",
			Capacity: api.NewResource(map[string]float64{"gpu": 4}),
			Labels:   map[string]string{"fabric-id": "fabric-a", "gpu-model": "c550"},
		},
		{
			Name:     "fabric-b-node",
			Capacity: api.NewResource(map[string]float64{"gpu": 4}),
			Labels:   map[string]string{"fabric-id": "fabric-b", "gpu-model": "c550"},
		},
	}
	job := api.Job{
		Name:         "small",
		MinAvailable: 2,
		Replicas:     2,
		Request:      api.NewResource(map[string]float64{"gpu": 1}),
		Topology:     &api.Topology{GPUModel: "c550", SameFabric: "Required"},
	}

	plan := New(nodes).Run([]api.Job{job})
	if len(plan.Allocations) != 2 {
		t.Fatalf("allocation count = %d, want 2", len(plan.Allocations))
	}
	for _, allocation := range plan.Allocations {
		if allocation.NodeName != "fabric-a-node" {
			t.Fatalf("allocation used %q, want fabric-a-node", allocation.NodeName)
		}
	}
}

func TestRunAllowsRequiredGangWhenFabricFitsMinAvailable(t *testing.T) {
	nodes := []api.Node{{
		Name:     "fabric-a-node",
		Capacity: api.NewResource(map[string]float64{"gpu": 4}),
		Labels:   map[string]string{"fabric-id": "fabric-a", "gpu-model": "c550"},
	}}
	job := api.Job{
		Name:         "elastic",
		Replicas:     8,
		MinAvailable: 4,
		Request:      api.NewResource(map[string]float64{"gpu": 1}),
		Topology:     &api.Topology{GPUModel: "c550", SameFabric: "Required"},
	}

	plan := New(nodes).Run([]api.Job{job})
	if len(plan.Allocations) != 4 {
		t.Fatalf("allocation count = %d, want 4", len(plan.Allocations))
	}
}

func TestRunSchedulesHigherWeightQueueFirst(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}}
	queues := []api.Queue{
		{Name: "low", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "high", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 1})},
	}
	jobs := []api.Job{
		{Name: "low-job", Queue: "low", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "high-job", Queue: "high", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	plan := New(nodes).RunWithQueues(jobs, queues)
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "high-job" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestRunWithQueuesRejectsGangExceedingCapability(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 8})}}
	queues := []api.Queue{{Name: "limited", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4})}}
	jobs := []api.Job{{Name: "large", Queue: "limited", Replicas: 8, MinAvailable: 8, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithQueues(jobs, queues)
	if len(plan.Allocations) != 0 || plan.Unschedulable["large"] != "queue capability exceeded" { t.Fatalf("unexpected plan: %#v", plan) }
}

func TestRunWithQueuesStopsAtCapabilityAfterGangIsReady(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 8})}}
	queues := []api.Queue{{Name: "limited", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4})}}
	jobs := []api.Job{{Name: "elastic", Queue: "limited", Replicas: 8, MinAvailable: 4, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithQueues(jobs, queues)
	if len(plan.Allocations) != 4 { t.Fatalf("allocation count = %d, want 4", len(plan.Allocations)) }
}
