package scheduler

import (
	"sync"
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

func TestSessionControllerCommitsAllocationIntoRunningTasks(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		[]*api.Job{{Name: "job", Queue: "q", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}},
		nil,
	)
	plan := controller.Run()
	if len(plan.Allocations) != 1 {
		t.Fatalf("initial plan = %#v", plan)
	}
	if err := controller.Commit(plan); err != nil {
		t.Fatal(err)
	}
	if next := controller.Run(); len(next.Allocations) != 0 {
		t.Fatalf("committed allocation was repeated: %#v", next)
	}
}

func TestSessionControllerCommitIsAtomic(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		[]*api.Job{{Name: "old", Queue: "q", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.RunningTask{{JobName: "old", QueueName: "q", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})}},
	)
	plan := api.AllocationPlan{
		Evictions:   []api.Eviction{{JobName: "old", TaskIndex: 0, NodeName: "n1"}},
		Allocations: []api.Allocation{{JobName: "missing", TaskIndex: 0, NodeName: "n1"}},
	}
	if err := controller.Commit(plan); err == nil {
		t.Fatal("invalid mixed plan was accepted")
	}
	if next := controller.Run(); len(next.Allocations) != 0 {
		t.Fatalf("partial commit released the old task: %#v", next)
	}
}

func TestSessionControllerRejectsRemovingBusyNode(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		nil,
		nil,
		[]api.RunningTask{{JobName: "job", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})}},
	)
	if err := controller.Apply(SessionEvent{Kind: EventRemoveNode, NodeName: "n1"}); err == nil {
		t.Fatal("busy node removal was accepted")
	}
	if err := controller.Apply(SessionEvent{Kind: EventUpdateRunningTasks}); err != nil {
		t.Fatal(err)
	}
	if err := controller.Apply(SessionEvent{Kind: EventRemoveNode, NodeName: "n1"}); err != nil {
		t.Fatalf("idle node removal failed: %v", err)
	}
}

func TestSessionControllerRejectsRemovingBusyJob(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		[]*api.Job{{Name: "job", Queue: "q", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.RunningTask{{JobName: "job", QueueName: "q", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})}},
	)
	if err := controller.Apply(SessionEvent{Kind: EventRemoveJob, JobName: "job"}); err == nil {
		t.Fatal("busy job removal was accepted")
	}
	if err := controller.Apply(SessionEvent{Kind: EventUpdateRunningTasks}); err != nil {
		t.Fatal(err)
	}
	if err := controller.Apply(SessionEvent{Kind: EventRemoveJob, JobName: "job"}); err != nil {
		t.Fatalf("idle job removal failed: %v", err)
	}
}

func TestSessionControllerSupportsConcurrentRuns(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		nil,
		nil,
		nil,
	)
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_ = controller.Run()
		}()
	}
	group.Wait()
}

func TestSessionControllerRejectsInvalidRunningTaskReferences(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}},
		[]*api.Job{{Name: "job", Queue: "q", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.RunningTask{{JobName: "job", QueueName: "q", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})}},
	)
	if err := controller.Apply(SessionEvent{Kind: EventUpdateRunningTasks, RunningTasks: []api.RunningTask{{JobName: "job", QueueName: "q", NodeName: "missing", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})}}}); err == nil {
		t.Fatal("invalid running task reference was accepted")
	}
	if plan := controller.Run(); len(plan.Allocations) != 0 {
		t.Fatalf("invalid update replaced the old snapshot: %#v", plan)
	}
}

func TestSessionControllerRejectsDuplicateRunningTasks(t *testing.T) {
	controller := NewSessionController(
		[]api.NodeSpec{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2})}},
		[]*api.Job{{Name: "job", Queue: "q", Replicas: 2, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}},
		[]api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 2})}},
		nil,
	)
	tasks := []api.RunningTask{
		{JobName: "job", QueueName: "q", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{JobName: "job", QueueName: "q", NodeName: "n1", TaskIndex: 0, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	if err := controller.Apply(SessionEvent{Kind: EventUpdateRunningTasks, RunningTasks: tasks}); err == nil {
		t.Fatal("duplicate running task was accepted")
	}
}
