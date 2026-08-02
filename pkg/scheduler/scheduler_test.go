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
