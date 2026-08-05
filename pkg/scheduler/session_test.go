package scheduler

import (
	"testing"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
)

func TestSessionControllerAppliesJobEventsBetweenRuns(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		nil,
		[]api.Queue{{Name: "default", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}},
		nil,
	)
	job := &api.Job{Name: "job", Queue: "default", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}
	if err := controller.Apply(SessionEvent{Kind: EventAddJob, Job: job}); err != nil {
		t.Fatal(err)
	}
	if plan := controller.Run(); len(plan.Allocations) != 1 {
		t.Fatalf("added job was not scheduled: %#v", plan)
	}
	if err := controller.Apply(SessionEvent{Kind: EventAddJob, Job: job}); err == nil {
		t.Fatal("duplicate job event was accepted")
	}
	if err := controller.Apply(SessionEvent{Kind: EventRemoveJob, JobName: "job"}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionControllerRejectsUnknownNodeRemoval(t *testing.T) {
	controller := NewSessionController(nil, nil, nil, nil)
	if err := controller.Apply(SessionEvent{Kind: EventRemoveNode, NodeName: "missing"}); err == nil {
		t.Fatal("unknown node removal was accepted")
	}
}

func TestSessionControllerRebuildsRuntimeStateFromRunningTasks(t *testing.T) {
	jobs := []*api.Job{
		{Name: "a-running", Queue: "q", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "z-pending", Queue: "q", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2})}},
		jobs,
		[]api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 2})}},
		[]api.RunningTask{{JobName: "a-running", QueueName: "q", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})}},
	)
	plan := controller.Run()
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "z-pending" {
		t.Fatalf("runtime state was not rebuilt: %#v", plan)
	}
}
