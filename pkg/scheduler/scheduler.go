package scheduler

import (
	"sort"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
	"github.com/zhouyingxiao/volcano-sim/pkg/proportion"
)

// Scheduler owns mutable node-idle state for one in-memory scheduling session.
type Scheduler struct {
	nodes []api.Node
	total api.Resource
}

// New creates a scheduler whose node state is independent of the caller.
func New(nodes []api.Node) *Scheduler {
	cloned := make([]api.Node, len(nodes))
	for i, node := range nodes {
		cloned[i] = node
		cloned[i].Capacity = api.NewResource(node.Capacity)
		if node.Idle == nil {
			cloned[i].Idle = api.NewResource(node.Capacity)
		} else {
			cloned[i].Idle = api.NewResource(node.Idle)
		}
	}
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].Name < cloned[j].Name })
	return &Scheduler{nodes: cloned, total: api.ClusterTotal(cloned)}
}

// OrderJobs returns a sorted copy, preserving the caller's input slice.
func (s *Scheduler) OrderJobs(jobs []*api.Job) []*api.Job {
	ordered := append([]*api.Job(nil), jobs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i].Allocated.DominantShare(s.total)
		right := ordered[j].Allocated.DominantShare(s.total)
		if left == right {
			return ordered[i].Name < ordered[j].Name
		}
		return left < right
	})
	return ordered
}

type trialAllocation struct {
	allocation api.Allocation
	nodeIndex  int
	request    api.Resource
}

// Run schedules each job as one gang transaction. It commits all successful
// trial allocations only when the transaction reaches MinAvailable.
func (s *Scheduler) Run(jobs []*api.Job) api.AllocationPlan {
	return s.runOrdered(s.OrderJobs(jobs), nil)
}

// RunWithQueues orders queues by weight, then preserves DRF ordering within each queue.
// The queues slice is session state: its order and Allocated fields may change.
func (s *Scheduler) RunWithQueues(jobs []*api.Job, queues []api.Queue) api.AllocationPlan {
	plan := api.AllocationPlan{Unschedulable: map[string]string{}}
	knownQueues := make(map[string]struct{}, len(queues))
	duplicateQueues := make(map[string]struct{})
	for _, queue := range queues {
		if _, exists := knownQueues[queue.Name]; exists {
			duplicateQueues[queue.Name] = struct{}{}
		}
		knownQueues[queue.Name] = struct{}{}
	}
	byQueue := make(map[string][]*api.Job)
	for _, job := range jobs {
		if _, duplicated := duplicateQueues[job.Queue]; duplicated {
			plan.Unschedulable[job.Name] = "queue duplicated"
			continue
		}
		if _, ok := knownQueues[job.Queue]; !ok {
			plan.Unschedulable[job.Name] = "queue not found"
			continue
		}
		byQueue[job.Queue] = append(byQueue[job.Queue], job)
	}
	deserved := proportion.ComputeDeserved(queues, s.total)
	for {
		bestIndex := -1
		for index := range queues {
			if len(byQueue[queues[index].Name]) == 0 {
				continue
			}
			if bestIndex < 0 || queueComesBefore(queues[index], queues[bestIndex], deserved, s.total) {
				bestIndex = index
			}
		}
		if bestIndex < 0 {
			break
		}
		queue := &queues[bestIndex]
		pending := byQueue[queue.Name]
		job := s.OrderJobs(pending)[0]
		partial := s.runOrdered([]*api.Job{job}, queue)
		plan.Allocations = append(plan.Allocations, partial.Allocations...)
		for name, reason := range partial.Unschedulable {
			plan.Unschedulable[name] = reason
		}
		if len(partial.Allocations) == 0 || job.ScheduledReplicas >= job.Replicas {
			for index, candidate := range pending {
				if candidate == job {
					byQueue[queue.Name] = append(pending[:index], pending[index+1:]...)
					break
				}
			}
		}
	}
	return plan
}

func queueComesBefore(left, right api.Queue, deserved map[string]api.Resource, total api.Resource) bool {
	leftDeficit := proportion.Deficit(left, deserved[left.Name], total)
	rightDeficit := proportion.Deficit(right, deserved[right.Name], total)
	if leftDeficit != rightDeficit {
		return leftDeficit > rightDeficit
	}
	if left.Weight != right.Weight {
		return left.Weight > right.Weight
	}
	return left.Name < right.Name
}

// RunSession keeps scheduling incomplete jobs until neither normal placement
// nor reclaim can make progress in the current session.
func (s *Scheduler) RunSession(jobs []*api.Job, queues []api.Queue, victims []api.RunningTask) api.AllocationPlan {
	return s.RunSessionDetailed(jobs, queues, victims).Summary()
}

// RunSessionDetailed preserves the boundaries between normal and reclaim rounds.
func (s *Scheduler) RunSessionDetailed(jobs []*api.Job, queues []api.Queue, victims []api.RunningTask) api.SessionPlan {
	plan := api.SessionPlan{}
	remainingVictims := append([]api.RunningTask(nil), victims...)
	for {
		pending := make([]*api.Job, 0, len(jobs))
		for _, job := range jobs {
			if job.ScheduledReplicas < job.Replicas {
				pending = append(pending, job)
			}
		}
		if len(pending) == 0 {
			break
		}

		normal := s.RunWithQueues(pending, queues)
		plan.Rounds = append(plan.Rounds, api.RoundPlan{Kind: "normal", Allocations: normal.Allocations, Evictions: normal.Evictions, Unschedulable: normal.Unschedulable})
		if len(normal.Allocations) > 0 {
			continue
		}

		reclaimed := api.AllocationPlan{Unschedulable: make(map[string]string)}
		for _, candidate := range s.OrderJobs(pending) {
			attempt := s.RunWithReclaim([]*api.Job{candidate}, queues, remainingVictims)
			reclaimed.Allocations = append(reclaimed.Allocations, attempt.Allocations...)
			reclaimed.Evictions = append(reclaimed.Evictions, attempt.Evictions...)
			for name, reason := range attempt.Unschedulable {
				reclaimed.Unschedulable[name] = reason
			}
			if len(attempt.Allocations) > 0 {
				break
			}
		}
		plan.Rounds = append(plan.Rounds, api.RoundPlan{Kind: "reclaim", Allocations: reclaimed.Allocations, Evictions: reclaimed.Evictions, Unschedulable: reclaimed.Unschedulable})
		if len(reclaimed.Allocations) == 0 {
			break
		}
		remainingVictims = removeEvictedVictims(remainingVictims, reclaimed.Evictions)
	}
	return plan
}

func removeEvictedVictims(victims []api.RunningTask, evictions []api.Eviction) []api.RunningTask {
	if len(evictions) == 0 {
		return victims
	}
	remaining := make([]api.RunningTask, 0, len(victims))
	for _, victim := range victims {
		removed := false
		for _, eviction := range evictions {
			if victim.JobName == eviction.JobName && victim.TaskIndex == eviction.TaskIndex && victim.NodeName == eviction.NodeName {
				removed = true
				break
			}
		}
		if !removed {
			remaining = append(remaining, victim)
		}
	}
	return remaining
}

// RunWithReclaim dry-runs victim evictions before attempting waiting gangs.
// Evictions are retained only when the subsequent allocation succeeds.
// Successful reclaim persists source-queue Allocated changes in queues; failed
// reclaim restores them before returning.
func (s *Scheduler) RunWithReclaim(jobs []*api.Job, queues []api.Queue, victims []api.RunningTask) api.AllocationPlan {
	plan := api.AllocationPlan{Unschedulable: make(map[string]string)}
	if len(jobs) == 0 {
		return plan
	}
	target := s.OrderJobs(jobs)[0]
	sort.SliceStable(victims, func(i, j int) bool {
		if victims[i].Priority != victims[j].Priority {
			return victims[i].Priority < victims[j].Priority
		}
		if victims[i].JobName != victims[j].JobName {
			return victims[i].JobName < victims[j].JobName
		}
		return victims[i].TaskIndex < victims[j].TaskIndex
	})
	deserved := proportion.ComputeDeserved(queues, s.total)
	queueIndex := make(map[string]int, len(queues))
	for index := range queues {
		queueIndex[queues[index].Name] = index
	}

	priorityBlocked := false
	need := api.NewResource(nil)
	remainingMin := target.MinAvailable - target.ScheduledReplicas
	if remainingMin < 0 {
		remainingMin = 0
	}
	for replica := 0; replica < remainingMin; replica++ {
		need = need.Add(target.Request)
	}
	available := api.NewResource(nil)
	for _, node := range s.nodes {
		available = available.Add(node.Idle)
	}
	for _, victim := range victims {
		if need.Fits(available) {
			break
		}
		if target.Priority <= victim.Priority {
			priorityBlocked = true
			continue
		}
		index, ok := queueIndex[victim.QueueName]
		if !ok || !queues[index].Reclaimable || !proportion.IsOverused(queues[index], deserved[victim.QueueName]) {
			continue
		}
		for nodeIndex := range s.nodes {
			if s.nodes[nodeIndex].Name == victim.NodeName {
				s.nodes[nodeIndex].Idle = s.nodes[nodeIndex].Idle.Add(victim.Request)
				available = available.Add(victim.Request)
				queues[index].Allocated = queues[index].Allocated.Sub(victim.Request)
				plan.Evictions = append(plan.Evictions, api.Eviction{JobName: victim.JobName, TaskIndex: victim.TaskIndex, NodeName: victim.NodeName, Reason: "reclaim"})
				break
			}
		}
	}

	allocated := s.RunWithQueues([]*api.Job{target}, queues)
	if len(allocated.Allocations) == 0 {
		if priorityBlocked {
			allocated.Unschedulable[target.Name] = "reclaim blocked by priority"
		}
		for _, eviction := range plan.Evictions {
			for _, victim := range victims {
				if victim.JobName == eviction.JobName && victim.TaskIndex == eviction.TaskIndex && victim.NodeName == eviction.NodeName {
					for index := range queues {
						if queues[index].Name == victim.QueueName {
							queues[index].Allocated = queues[index].Allocated.Add(victim.Request)
							break
						}
					}
					for nodeIndex := range s.nodes {
						if s.nodes[nodeIndex].Name == victim.NodeName {
							s.nodes[nodeIndex].Idle = s.nodes[nodeIndex].Idle.Sub(victim.Request)
						}
					}
				}
			}
		}
		return allocated
	}
	allocated.Evictions = plan.Evictions
	return allocated
}

func (s *Scheduler) runOrdered(jobs []*api.Job, queue *api.Queue) api.AllocationPlan {
	plan := api.AllocationPlan{Unschedulable: make(map[string]string)}
	for _, job := range jobs {
		candidates, reason := s.selectCandidateNodes(*job)
		if reason != "" {
			plan.Unschedulable[job.Name] = reason
			continue
		}
		journal := make([]trialAllocation, 0, job.Replicas)
		journalResource := api.NewResource(nil)
		capabilityBlocked := false
		batchLimit := job.BatchSize
		if batchLimit == 0 {
			batchLimit = job.Replicas
		}
		if job.ScheduledReplicas == 0 && batchLimit < job.MinAvailable {
			batchLimit = job.MinAvailable
		}
		remaining := job.Replicas - job.ScheduledReplicas
		if batchLimit > remaining {
			batchLimit = remaining
		}
		for taskIndex := 0; taskIndex < batchLimit; taskIndex++ {
			if queue != nil && !queue.Allocated.Add(journalResource).Add(job.Request).Fits(queue.Capability) {
				capabilityBlocked = true
				break
			}
			nodeIndex := s.findNode(job.Request, candidates)
			if nodeIndex < 0 {
				break
			}
			s.nodes[nodeIndex].Idle = s.nodes[nodeIndex].Idle.Sub(job.Request)
			journal = append(journal, trialAllocation{
				allocation: api.Allocation{JobName: job.Name, TaskIndex: job.ScheduledReplicas + taskIndex, NodeName: s.nodes[nodeIndex].Name},
				nodeIndex:  nodeIndex,
				request:    job.Request,
			})
			journalResource = journalResource.Add(job.Request)
		}

		if job.ScheduledReplicas+len(journal) < job.MinAvailable {
			for index := len(journal) - 1; index >= 0; index-- {
				trial := journal[index]
				s.nodes[trial.nodeIndex].Idle = s.nodes[trial.nodeIndex].Idle.Add(trial.request)
			}
			if capabilityBlocked {
				plan.Unschedulable[job.Name] = "queue capability exceeded"
			} else {
				plan.Unschedulable[job.Name] = "insufficient resources for minAvailable"
			}
			continue
		}
		for _, trial := range journal {
			plan.Allocations = append(plan.Allocations, trial.allocation)
		}
		job.ScheduledReplicas += len(journal)
		job.Allocated = job.Allocated.Add(journalResource)
		if queue != nil {
			queue.Allocated = queue.Allocated.Add(journalResource)
		}
	}
	return plan
}

// selectCandidateNodes chooses one complete matching Fabric before any Gang
// transaction changes node idle resources.
func (s *Scheduler) selectCandidateNodes(job api.Job) ([]int, string) {
	if job.Topology == nil || job.Topology.SameFabric == "" {
		candidates := make([]int, len(s.nodes))
		for index := range s.nodes {
			candidates[index] = index
		}
		return candidates, ""
	}

	need := api.NewResource(nil)
	for replica := 0; replica < job.MinAvailable; replica++ {
		need = need.Add(job.Request)
	}
	fabrics := make(map[string][]int)
	for index, node := range s.nodes {
		if node.Labels["gpu-model"] != job.Topology.GPUModel || node.Labels["fabric-id"] == "" {
			continue
		}
		fabricID := node.Labels["fabric-id"]
		fabrics[fabricID] = append(fabrics[fabricID], index)
	}

	fabricIDs := make([]string, 0, len(fabrics))
	for fabricID := range fabrics {
		fabricIDs = append(fabricIDs, fabricID)
	}
	sort.Strings(fabricIDs)
	for _, fabricID := range fabricIDs {
		idle := api.NewResource(nil)
		for _, index := range fabrics[fabricID] {
			idle = idle.Add(s.nodes[index].Idle)
		}
		if need.Fits(idle) {
			return fabrics[fabricID], ""
		}
	}
	return nil, "no single fabric has enough capacity"
}

func (s *Scheduler) findNode(request api.Resource, candidates []int) int {
	for _, index := range candidates {
		node := s.nodes[index]
		if request.Fits(node.Idle) {
			return index
		}
	}
	return -1
}
