package scheduler

import (
	"testing"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
)

func jobPtrs(jobs []api.Job) []*api.Job {
	result := make([]*api.Job, len(jobs))
	for index := range jobs {
		result[index] = &jobs[index]
	}
	return result
}

func TestOrderJobsUsesLowestDominantShareFirst(t *testing.T) {
	scheduler := New([]api.Node{{
		Name:     "n",
		Capacity: api.NewResource(map[string]float64{"cpu": 100, "gpu": 10}),
	}})
	jobs := []api.Job{
		{Name: "gpu-heavy", Allocated: api.NewResource(map[string]float64{"gpu": 5})},
		{Name: "cpu-heavy", Allocated: api.NewResource(map[string]float64{"cpu": 20})},
	}

	got := scheduler.OrderJobs(jobPtrs(jobs))
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

	plan := scheduler.Run(jobPtrs([]api.Job{{
		Name:         "four-gpu-tasks",
		MinAvailable: 4,
		Replicas:     4,
		Request:      api.NewResource(map[string]float64{"gpu": 1}),
	}}))

	if len(plan.Allocations) != 0 {
		t.Fatalf("partial gang allocation leaked: %#v", plan.Allocations)
	}
	if plan.Unschedulable["four-gpu-tasks"] != "insufficient resources for minAvailable" {
		t.Fatalf("missing unschedulable reason: %#v", plan.Unschedulable)
	}
}

func TestRunUpdatesJobStateAfterCommit(t *testing.T) {
	job := &api.Job{Name: "stateful", Replicas: 2, MinAvailable: 2, BatchSize: 2, Request: api.NewResource(map[string]float64{"gpu": 1}), Allocated: api.NewResource(nil)}
	New([]api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2})}}).Run([]*api.Job{job})
	if job.ScheduledReplicas != 2 || job.Allocated["gpu"] != 2 {
		t.Fatalf("job state was not updated: %#v", job)
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

	plan := New(nodes).Run(jobPtrs([]api.Job{job}))
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

	plan := New(nodes).Run(jobPtrs([]api.Job{job}))
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

	plan := New(nodes).Run(jobPtrs([]api.Job{job}))
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
	plan := New(nodes).RunWithQueues(jobPtrs(jobs), queues)
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "high-job" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestRunWithQueuesUsesProportionDeficitAcrossQueues(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 4})}}
	queues := []api.Queue{
		{Name: "small", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4})},
		{Name: "large", Weight: 3, Capability: api.NewResource(map[string]float64{"gpu": 4})},
	}
	jobs := []*api.Job{
		{Name: "small-job", Queue: "small", Replicas: 4, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "large-job", Queue: "large", Replicas: 4, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	plan := New(nodes).RunWithQueues(jobs, queues)
	counts := map[string]int{}
	for _, allocation := range plan.Allocations {
		counts[allocation.JobName]++
	}
	if counts["small-job"] != 1 || counts["large-job"] != 3 {
		t.Fatalf("unexpected proportion allocation: %#v", plan.Allocations)
	}
}

func TestRunWithQueuesRejectsGangExceedingCapability(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 8})}}
	queues := []api.Queue{{Name: "limited", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4})}}
	jobs := []api.Job{{Name: "large", Queue: "limited", Replicas: 8, MinAvailable: 8, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithQueues(jobPtrs(jobs), queues)
	if len(plan.Allocations) != 0 || plan.Unschedulable["large"] != "queue capability exceeded" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestReclaimEvictsOverusedReclaimableTaskForWaitingGang(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "inference", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
	}
	victims := []api.RunningTask{{JobName: "old", QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []api.Job{{Name: "new", Priority: 100, Queue: "inference", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithReclaim(jobPtrs(jobs), queues, victims)
	if len(plan.Evictions) != 1 || len(plan.Allocations) != 1 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	var trainingAllocated float64
	for _, queue := range queues {
		if queue.Name == "training" {
			trainingAllocated = queue.Allocated["gpu"]
		}
	}
	if trainingAllocated != 0 {
		t.Fatalf("successful reclaim did not persist source queue state: %#v", queues)
	}
}

func TestReclaimEvictsLowerPriorityVictimFirst(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 2}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 2})}, {Name: "inference", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 2}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})}}
	victims := []api.RunningTask{{JobName: "important", Priority: 100, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}, {JobName: "batch", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []api.Job{{Name: "new", Priority: 100, Queue: "inference", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithReclaim(jobPtrs(jobs), queues, victims)
	if len(plan.Evictions) != 1 || plan.Evictions[0].JobName != "batch" {
		t.Fatalf("unexpected evictions: %#v", plan.Evictions)
	}
}

func TestReclaimDoesNotEvictHigherPriorityVictim(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})}, {Name: "inference", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})}}
	victims := []api.RunningTask{{JobName: "important", Priority: 100, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []api.Job{{Name: "low", Priority: 10, Queue: "inference", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithReclaim(jobPtrs(jobs), queues, victims)
	if len(plan.Evictions) != 0 || len(plan.Allocations) != 0 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if plan.Unschedulable["low"] != "reclaim blocked by priority" {
		t.Fatalf("unexpected reason: %#v", plan.Unschedulable)
	}
}

func TestReclaimRollsBackEvictionWhenGangIsStillNotReady(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})}, {Name: "inference", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 2}), Guarantee: api.NewResource(map[string]float64{"gpu": 2})}}
	victims := []api.RunningTask{{JobName: "old", QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []api.Job{{Name: "new", Priority: 100, Queue: "inference", Replicas: 2, MinAvailable: 2, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithReclaim(jobPtrs(jobs), queues, victims)
	if len(plan.Evictions) != 0 || len(plan.Allocations) != 0 {
		t.Fatalf("reclaim leaked into failed plan: %#v", plan)
	}
	for _, queue := range queues {
		if queue.Name == "training" && queue.Allocated["gpu"] != 1 {
			t.Fatalf("source queue allocation was not restored: %#v", queues)
		}
		if queue.Name == "inference" && queue.Allocated["gpu"] != 0 {
			t.Fatalf("unrelated queue allocation changed: %#v", queues)
		}
	}
}

func TestReclaimEvictsOnlyVictimsNeededForGang(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 3}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 3}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 3})}, {Name: "inference", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 3}), Guarantee: api.NewResource(map[string]float64{"gpu": 2})}}
	victims := []api.RunningTask{{JobName: "old", QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}, {JobName: "old", QueueName: "training", TaskIndex: 1, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}, {JobName: "old", QueueName: "training", TaskIndex: 2, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []api.Job{{Name: "new", Priority: 100, Queue: "inference", Replicas: 2, MinAvailable: 2, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithReclaim(jobPtrs(jobs), queues, victims)
	if len(plan.Evictions) != 2 || len(plan.Allocations) != 2 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestReclaimStopsWhenGangNeedIsMetEvenIfSourceRemainsOverused(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 3}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 3}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 3})}, {Name: "inference", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 3}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})}}
	victims := []api.RunningTask{{JobName: "old", QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}, {JobName: "old", QueueName: "training", TaskIndex: 1, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}, {JobName: "old", QueueName: "training", TaskIndex: 2, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []api.Job{{Name: "new", Priority: 100, Queue: "inference", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithReclaim(jobPtrs(jobs), queues, victims)
	if len(plan.Evictions) != 1 || len(plan.Allocations) != 1 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestRunWithQueuesStopsAtCapabilityAfterGangIsReady(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 8})}}
	queues := []api.Queue{{Name: "limited", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4})}}
	jobs := []api.Job{{Name: "elastic", Queue: "limited", Replicas: 8, MinAvailable: 4, Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunWithQueues(jobPtrs(jobs), queues)
	if len(plan.Allocations) != 4 {
		t.Fatalf("allocation count = %d, want 4", len(plan.Allocations))
	}
}

func TestRunWithQueuesUsesContinuousTaskIndexesAcrossBatches(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2})}}
	queues := []api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 2})}}
	job := &api.Job{Name: "batched", Queue: "q", Replicas: 2, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1})}
	plan := New(nodes).RunWithQueues([]*api.Job{job}, queues)
	if len(plan.Allocations) != 2 {
		t.Fatalf("allocation count = %d, want 2", len(plan.Allocations))
	}
	if plan.Allocations[0].TaskIndex != 0 || plan.Allocations[1].TaskIndex != 1 {
		t.Fatalf("task indexes = %#v, want [0 1]", plan.Allocations)
	}
}

func TestRunWithQueuesReordersIncompleteJobsBetweenBatches(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 4})}}
	queues := []api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4})}}
	a := &api.Job{Name: "a", Queue: "q", Replicas: 4, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Allocated: api.NewResource(nil)}
	b := &api.Job{Name: "b", Queue: "q", Replicas: 4, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Allocated: api.NewResource(nil)}
	plan := New(nodes).RunWithQueues([]*api.Job{a, b}, queues)
	if len(plan.Allocations) != 4 {
		t.Fatalf("allocation count = %d, want 4", len(plan.Allocations))
	}
	for index, want := range []string{"a", "b", "a", "b"} {
		if plan.Allocations[index].JobName != want {
			t.Fatalf("allocation %d = %q, want %q", index, plan.Allocations[index].JobName, want)
		}
	}
}

func TestRunWithQueuesStopsWhenNoPendingJobCanProgress(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "q", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}}
	job := &api.Job{Name: "blocked", Queue: "q", Replicas: 1, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Allocated: api.NewResource(nil)}
	plan := New(nodes).RunWithQueues([]*api.Job{job}, queues)
	if len(plan.Allocations) != 0 || plan.Unschedulable["blocked"] == "" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestRunWithQueuesReportsUnknownQueue(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}}
	queues := []api.Queue{{Name: "known", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []*api.Job{
		{Name: "missing-queue-job", Queue: "missing", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "known-queue-job", Queue: "known", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	plan := New(nodes).RunWithQueues(jobs, queues)
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "known-queue-job" {
		t.Fatalf("unexpected allocations: %#v", plan.Allocations)
	}
	if plan.Unschedulable["missing-queue-job"] != "queue not found" {
		t.Fatalf("unexpected admission reason: %#v", plan.Unschedulable)
	}
}

func TestRunWithQueuesRejectsDuplicateQueueNames(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2})}}
	queues := []api.Queue{
		{Name: "duplicate", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "duplicate", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "known", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1})},
	}
	jobs := []*api.Job{
		{Name: "duplicate-job", Queue: "duplicate", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "known-job", Queue: "known", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	plan := New(nodes).RunWithQueues(jobs, queues)
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "known-job" {
		t.Fatalf("unexpected allocations: %#v", plan.Allocations)
	}
	if plan.Unschedulable["duplicate-job"] != "queue duplicated" {
		t.Fatalf("unexpected duplicate queue reason: %#v", plan.Unschedulable)
	}
}

func TestRunSessionReclaimsWhenNormalBatchCannotProgress(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})}, {Name: "inference", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})}}
	victims := []api.RunningTask{{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	job := &api.Job{Name: "new", Priority: 100, Queue: "inference", Replicas: 1, MinAvailable: 1, BatchSize: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Allocated: api.NewResource(nil)}
	plan := New(nodes).RunSession([]*api.Job{job}, queues, victims)
	if len(plan.Evictions) != 1 || len(plan.Allocations) != 1 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestRunSessionContinuesAfterNormalProgressToReclaimAnotherJob(t *testing.T) {
	nodes := []api.Node{
		{Name: "a", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Labels: map[string]string{"fabric-id": "fa", "gpu-model": "m"}},
		{Name: "b", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{}), Labels: map[string]string{"fabric-id": "fb", "gpu-model": "m"}},
	}
	queues := []api.Queue{
		{Name: "first", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "waiting", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})},
	}
	jobs := []*api.Job{
		{Name: "first-job", Priority: 100, Queue: "first", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "waiting-job", Priority: 200, Queue: "waiting", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Topology: &api.Topology{GPUModel: "m", SameFabric: "Required"}},
	}
	victims := []api.RunningTask{{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "b", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunSession(jobs, queues, victims)
	if len(plan.Allocations) != 2 || len(plan.Evictions) != 1 {
		t.Fatalf("unexpected session plan: %#v", plan)
	}
}

func TestRunSessionDetailedPreservesRoundBoundaries(t *testing.T) {
	nodes := []api.Node{
		{Name: "a", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Labels: map[string]string{"fabric-id": "fa", "gpu-model": "m"}},
		{Name: "b", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{}), Labels: map[string]string{"fabric-id": "fb", "gpu-model": "m"}},
	}
	queues := []api.Queue{
		{Name: "first", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "waiting", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})},
	}
	jobs := []*api.Job{
		{Name: "first-job", Priority: 100, Queue: "first", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "waiting-job", Priority: 200, Queue: "waiting", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Topology: &api.Topology{GPUModel: "m", SameFabric: "Required"}},
	}
	victims := []api.RunningTask{{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "b", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	plan := New(nodes).RunSessionDetailed(jobs, queues, victims)
	if len(plan.Rounds) != 3 || plan.Rounds[0].Kind != "normal" || plan.Rounds[1].Kind != "normal" || plan.Rounds[2].Kind != "reclaim" {
		t.Fatalf("unexpected rounds: %#v", plan.Rounds)
	}
	if summary := plan.Summary(); len(summary.Allocations) != 2 || len(summary.Evictions) != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestRunWithReclaimSelectsTargetByDRFOrder(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{
		{Name: "blocked", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "ready", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})},
	}
	victims := []api.RunningTask{{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []*api.Job{
		{Name: "blocked-job", Priority: 10, Queue: "blocked", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1}), Allocated: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "ready-job", Priority: 100, Queue: "ready", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	plan := New(nodes).RunWithReclaim(jobs, queues, victims)
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "ready-job" {
		t.Fatalf("unexpected target allocation: %#v", plan)
	}
}

func TestRunWithReclaimAcceptsEmptyJobList(t *testing.T) {
	plan := New([]api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1})}}).RunWithReclaim(nil, nil, nil)
	if len(plan.Allocations) != 0 || len(plan.Evictions) != 0 {
		t.Fatalf("unexpected empty reclaim plan: %#v", plan)
	}
}

func TestRunWithReclaimOnlyFreesRemainingMinAvailable(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 4}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 4}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 4})},
		{Name: "inference", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 4}), Guarantee: api.NewResource(map[string]float64{"gpu": 4}), Allocated: api.NewResource(map[string]float64{"gpu": 2})},
	}
	victims := make([]api.RunningTask, 4)
	for index := range victims {
		victims[index] = api.RunningTask{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: index, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}
	}
	job := &api.Job{Name: "partial", Priority: 100, Queue: "inference", Replicas: 4, MinAvailable: 4, ScheduledReplicas: 2, Allocated: api.NewResource(map[string]float64{"gpu": 2}), Request: api.NewResource(map[string]float64{"gpu": 1})}
	plan := New(nodes).RunWithReclaim([]*api.Job{job}, queues, victims)
	if len(plan.Evictions) != 2 || len(plan.Allocations) != 2 {
		t.Fatalf("unexpected partial reclaim plan: %#v", plan)
	}
}

func TestReclaimPreservesVictimJobMinAvailable(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 3}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 3}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 3})},
		{Name: "inference", Weight: 2, Capability: api.NewResource(map[string]float64{"gpu": 3}), Guarantee: api.NewResource(map[string]float64{"gpu": 2})},
	}
	victims := []api.RunningTask{
		{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1}), JobMinAvailable: 2, JobRunningReplicas: 3},
		{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 1, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1}), JobMinAvailable: 2, JobRunningReplicas: 3},
		{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 2, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1}), JobMinAvailable: 2, JobRunningReplicas: 3},
	}
	job := &api.Job{Name: "new", Priority: 100, Queue: "inference", Replicas: 2, MinAvailable: 2, Request: api.NewResource(map[string]float64{"gpu": 1})}
	plan := New(nodes).RunWithReclaim([]*api.Job{job}, queues, victims)
	if len(plan.Allocations) != 0 || len(plan.Evictions) != 0 {
		t.Fatalf("reclaim violated victim gang: %#v", plan)
	}
}

func TestRunSessionTriesNextReclaimCandidateAfterFailure(t *testing.T) {
	nodes := []api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 1}), Idle: api.NewResource(map[string]float64{})}}
	queues := []api.Queue{
		{Name: "a-queue", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "b-queue", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Guarantee: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "training", Weight: 1, Capability: api.NewResource(map[string]float64{"gpu": 1}), Reclaimable: true, Allocated: api.NewResource(map[string]float64{"gpu": 1})},
	}
	victims := []api.RunningTask{{JobName: "old", Priority: 10, QueueName: "training", TaskIndex: 0, NodeName: "n1", Request: api.NewResource(map[string]float64{"gpu": 1})}}
	jobs := []*api.Job{
		{Name: "a-job", Priority: 10, Queue: "a-queue", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
		{Name: "b-job", Priority: 100, Queue: "b-queue", Replicas: 1, MinAvailable: 1, Request: api.NewResource(map[string]float64{"gpu": 1})},
	}
	plan := New(nodes).RunSession(jobs, queues, victims)
	if len(plan.Allocations) != 1 || plan.Allocations[0].JobName != "b-job" || len(plan.Evictions) != 1 {
		t.Fatalf("unexpected candidate reclaim plan: %#v", plan)
	}
}
