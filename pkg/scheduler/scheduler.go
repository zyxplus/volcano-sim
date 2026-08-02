package scheduler

import (
	"sort"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
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
func (s *Scheduler) OrderJobs(jobs []api.Job) []api.Job {
	ordered := append([]api.Job(nil), jobs...)
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
func (s *Scheduler) Run(jobs []api.Job) api.AllocationPlan {
	plan := api.AllocationPlan{Unschedulable: make(map[string]string)}
	for _, job := range s.OrderJobs(jobs) {
		journal := make([]trialAllocation, 0, job.Replicas)
		for taskIndex := 0; taskIndex < job.Replicas; taskIndex++ {
			nodeIndex := s.findNode(job.Request)
			if nodeIndex < 0 {
				break
			}
			s.nodes[nodeIndex].Idle = s.nodes[nodeIndex].Idle.Sub(job.Request)
			journal = append(journal, trialAllocation{
				allocation: api.Allocation{JobName: job.Name, TaskIndex: taskIndex, NodeName: s.nodes[nodeIndex].Name},
				nodeIndex:  nodeIndex,
				request:    job.Request,
			})
		}

		if len(journal) < job.MinAvailable {
			for index := len(journal) - 1; index >= 0; index-- {
				trial := journal[index]
				s.nodes[trial.nodeIndex].Idle = s.nodes[trial.nodeIndex].Idle.Add(trial.request)
			}
			plan.Unschedulable[job.Name] = "insufficient resources for minAvailable"
			continue
		}
		for _, trial := range journal {
			plan.Allocations = append(plan.Allocations, trial.allocation)
		}
	}
	return plan
}

func (s *Scheduler) findNode(request api.Resource) int {
	for index, node := range s.nodes {
		if request.Fits(node.Idle) {
			return index
		}
	}
	return -1
}
