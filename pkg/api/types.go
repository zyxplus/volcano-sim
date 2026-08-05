package api

// Node is a schedulable machine. Idle changes during a scheduling session;
// Capacity remains the machine's total resource vector.
type Node struct {
	Name     string            `yaml:"name" json:"name"`
	Capacity Resource          `yaml:"capacity" json:"capacity"`
	Labels   map[string]string `yaml:"labels" json:"labels"`
	Idle     Resource          `yaml:"-" json:"idle"`
}

// NodeSpec is the immutable node configuration used to initialize sessions.
type NodeSpec struct {
	Name     string
	Capacity Resource
	Labels   map[string]string
}

// NodeState is the mutable state of one node inside a scheduling session.
type NodeState struct {
	Name     string
	Capacity Resource
	Labels   map[string]string
	Idle     Resource
}

// Topology constrains where a Job's task replicas may be placed.
type Topology struct {
	GPUModel   string `yaml:"gpuModel" json:"gpuModel"`
	SameFabric string `yaml:"sameFabric" json:"sameFabric"`
}

// Queue groups jobs under a weighted, resource-capped admission boundary.
type Queue struct {
	Name        string   `yaml:"name" json:"name"`
	Weight      int      `yaml:"weight" json:"weight"`
	Capability  Resource `yaml:"capability" json:"capability"`
	Guarantee   Resource `yaml:"guarantee" json:"guarantee"`
	Reclaimable bool     `yaml:"reclaimable" json:"reclaimable"`
	Allocated   Resource `yaml:"-" json:"allocated"`
}

// Job describes identical task replicas that must reach MinAvailable together.
type Job struct {
	Name              string    `yaml:"name" json:"name"`
	Priority          int       `yaml:"priority" json:"priority"`
	MinAvailable      int       `yaml:"minAvailable" json:"minAvailable"`
	Replicas          int       `yaml:"replicas" json:"replicas"`
	BatchSize         int       `yaml:"batchSize" json:"batchSize"`
	ScheduledReplicas int       `yaml:"-" json:"scheduledReplicas"`
	Request           Resource  `yaml:"request" json:"request"`
	Queue             string    `yaml:"queue" json:"queue"`
	Topology          *Topology `yaml:"topology" json:"topology,omitempty"`
	Allocated         Resource  `yaml:"-" json:"allocated"`
}

// Task identifies one replica of a Job during trial allocation.
type Task struct {
	JobName string   `json:"jobName"`
	Index   int      `json:"index"`
	Request Resource `json:"request"`
}

// RunningTask is an already placed task that may be selected as a reclaim victim.
type RunningTask struct {
	JobName   string
	Priority  int
	QueueName string
	TaskIndex int
	NodeName  string
	Request   Resource
	// Optional Job-level state used to protect a victim Gang during reclaim.
	JobMinAvailable    int
	JobRunningReplicas int
}

// Allocation records a committed placement decision.
type Allocation struct {
	JobName   string `json:"jobName"`
	TaskIndex int    `json:"taskIndex"`
	NodeName  string `json:"nodeName"`
}

// Eviction records a dry-run decision to release a running task's resources.
type Eviction struct {
	JobName   string `json:"jobName"`
	TaskIndex int    `json:"taskIndex"`
	NodeName  string `json:"nodeName"`
	Reason    string `json:"reason"`
}

// AllocationPlan is the scheduler's side-effect-free output.
type AllocationPlan struct {
	Allocations   []Allocation      `json:"allocations"`
	Evictions     []Eviction        `json:"evictions"`
	Unschedulable map[string]string `json:"unschedulable"`
}

// RoundPlan records the result of one normal-scheduling or reclaim attempt.
type RoundPlan struct {
	Kind          string            `json:"kind"`
	Allocations   []Allocation      `json:"allocations"`
	Evictions     []Eviction        `json:"evictions"`
	Unschedulable map[string]string `json:"unschedulable"`
}

// SessionPlan preserves round boundaries while allowing a legacy summary view.
type SessionPlan struct {
	Rounds []RoundPlan `json:"rounds"`
}

// Summary aggregates all round decisions into the legacy plan shape.
func (p SessionPlan) Summary() AllocationPlan {
	plan := AllocationPlan{Unschedulable: make(map[string]string)}
	for _, round := range p.Rounds {
		plan.Allocations = append(plan.Allocations, round.Allocations...)
		plan.Evictions = append(plan.Evictions, round.Evictions...)
		for name, reason := range round.Unschedulable {
			plan.Unschedulable[name] = reason
		}
	}
	return plan
}

// ClusterTotal sums the capacity of every node.
func ClusterTotal(nodes []Node) Resource {
	total := NewResource(nil)
	for _, node := range nodes {
		total = total.Add(node.Capacity)
	}
	return total
}
