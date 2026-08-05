package scheduler

import (
	"fmt"
	"sort"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
)

type EventKind string

const (
	EventAddJob             EventKind = "addJob"
	EventRemoveJob          EventKind = "removeJob"
	EventAddNode            EventKind = "addNode"
	EventRemoveNode         EventKind = "removeNode"
	EventUpdateRunningTasks EventKind = "updateRunningTasks"
)

// SessionEvent changes the input snapshot used by the next Run.
type SessionEvent struct {
	Kind         EventKind
	Job          *api.Job
	JobName      string
	Node         *api.NodeSpec
	NodeName     string
	RunningTasks []api.RunningTask
}

// SessionController owns mutable input snapshots across independent Sessions.
type SessionController struct {
	nodes   map[string]api.NodeSpec
	jobs    map[string]*api.Job
	queues  []api.Queue
	victims []api.RunningTask
}

func NewSessionController(nodes []api.NodeSpec, jobs []*api.Job, queues []api.Queue, victims []api.RunningTask) *SessionController {
	controller := &SessionController{nodes: make(map[string]api.NodeSpec), jobs: make(map[string]*api.Job), queues: cloneQueues(queues), victims: append([]api.RunningTask(nil), victims...)}
	for _, node := range cloneNodeSpecs(nodes) {
		controller.nodes[node.Name] = node
	}
	for _, job := range jobs {
		controller.jobs[job.Name] = cloneJob(job)
	}
	return controller
}

func (c *SessionController) Apply(event SessionEvent) error {
	switch event.Kind {
	case EventAddJob:
		if event.Job == nil || event.Job.Name == "" {
			return fmt.Errorf("add job requires a named job")
		}
		if _, exists := c.jobs[event.Job.Name]; exists {
			return fmt.Errorf("job %q already exists", event.Job.Name)
		}
		c.jobs[event.Job.Name] = cloneJob(event.Job)
	case EventRemoveJob:
		if _, exists := c.jobs[event.JobName]; !exists {
			return fmt.Errorf("job %q does not exist", event.JobName)
		}
		for _, task := range c.victims {
			if task.JobName == event.JobName {
				return fmt.Errorf("job %q still has running tasks", event.JobName)
			}
		}
		delete(c.jobs, event.JobName)
	case EventAddNode:
		if event.Node == nil || event.Node.Name == "" {
			return fmt.Errorf("add node requires a named node")
		}
		if _, exists := c.nodes[event.Node.Name]; exists {
			return fmt.Errorf("node %q already exists", event.Node.Name)
		}
		c.nodes[event.Node.Name] = cloneNodeSpecs([]api.NodeSpec{*event.Node})[0]
	case EventRemoveNode:
		if _, exists := c.nodes[event.NodeName]; !exists {
			return fmt.Errorf("node %q does not exist", event.NodeName)
		}
		for _, task := range c.victims {
			if task.NodeName == event.NodeName {
				return fmt.Errorf("node %q still has running tasks", event.NodeName)
			}
		}
		delete(c.nodes, event.NodeName)
	case EventUpdateRunningTasks:
		c.victims = append([]api.RunningTask(nil), event.RunningTasks...)
	default:
		return fmt.Errorf("unknown session event %q", event.Kind)
	}
	return nil
}

// Run creates a fresh Scheduler from the current input snapshot.
func (c *SessionController) Run() api.AllocationPlan {
	c.rebuildRuntimeState()
	nodes := make([]api.Node, 0, len(c.nodes))
	for _, spec := range c.nodes {
		nodes = append(nodes, api.Node{Name: spec.Name, Capacity: api.NewResource(spec.Capacity), Labels: spec.Labels, Idle: api.NewResource(spec.Capacity)})
	}
	for _, task := range c.victims {
		for index := range nodes {
			if nodes[index].Name == task.NodeName {
				nodes[index].Idle = nodes[index].Idle.Sub(task.Request)
				break
			}
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	jobs := make([]*api.Job, 0, len(c.jobs))
	for _, job := range c.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Name < jobs[j].Name })
	return New(nodes).RunSession(jobs, cloneQueues(c.queues), c.victims)
}

// Commit applies a plan to the in-memory running-task snapshot.
func (c *SessionController) Commit(plan api.AllocationPlan) error {
	working := append([]api.RunningTask(nil), c.victims...)
	for _, eviction := range plan.Evictions {
		removed := false
		remaining := make([]api.RunningTask, 0, len(working))
		for _, task := range working {
			if task.JobName == eviction.JobName && task.TaskIndex == eviction.TaskIndex && task.NodeName == eviction.NodeName {
				removed = true
				continue
			}
			remaining = append(remaining, task)
		}
		if !removed {
			return fmt.Errorf("eviction target %s/%d on %s does not exist", eviction.JobName, eviction.TaskIndex, eviction.NodeName)
		}
		working = remaining
	}
	for _, allocation := range plan.Allocations {
		job, ok := c.jobs[allocation.JobName]
		if !ok {
			return fmt.Errorf("allocation job %q does not exist", allocation.JobName)
		}
		for _, task := range working {
			if task.JobName == allocation.JobName && task.TaskIndex == allocation.TaskIndex && task.NodeName == allocation.NodeName {
				return fmt.Errorf("allocation target %s/%d on %s already exists", allocation.JobName, allocation.TaskIndex, allocation.NodeName)
			}
		}
		working = append(working, api.RunningTask{
			JobName:            job.Name,
			Priority:           job.Priority,
			QueueName:          job.Queue,
			TaskIndex:          allocation.TaskIndex,
			NodeName:           allocation.NodeName,
			Request:            api.NewResource(job.Request),
			JobMinAvailable:    job.MinAvailable,
			JobRunningReplicas: job.ScheduledReplicas,
		})
	}
	c.victims = working
	return nil
}

func (c *SessionController) rebuildRuntimeState() {
	for _, job := range c.jobs {
		job.ScheduledReplicas = 0
		job.Allocated = api.NewResource(nil)
	}
	for index := range c.queues {
		c.queues[index].Allocated = api.NewResource(nil)
	}
	for _, task := range c.victims {
		if job, ok := c.jobs[task.JobName]; ok {
			job.ScheduledReplicas++
			job.Allocated = job.Allocated.Add(task.Request)
		}
		for index := range c.queues {
			if c.queues[index].Name == task.QueueName {
				c.queues[index].Allocated = c.queues[index].Allocated.Add(task.Request)
				break
			}
		}
	}
}

func cloneQueues(queues []api.Queue) []api.Queue {
	cloned := make([]api.Queue, len(queues))
	for index, queue := range queues {
		cloned[index] = queue
		cloned[index].Capability = api.NewResource(queue.Capability)
		cloned[index].Guarantee = api.NewResource(queue.Guarantee)
		cloned[index].Allocated = api.NewResource(queue.Allocated)
	}
	return cloned
}

func cloneJob(job *api.Job) *api.Job {
	cloned := *job
	cloned.Request = api.NewResource(job.Request)
	cloned.Allocated = api.NewResource(job.Allocated)
	if job.Topology != nil {
		topology := *job.Topology
		cloned.Topology = &topology
	}
	return &cloned
}
