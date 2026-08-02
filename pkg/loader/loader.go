// Package loader converts YAML scenario files into validated scheduler input.
package loader

import (
	"fmt"
	"os"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
	"gopkg.in/yaml.v3"
)

type nodesDocument struct {
	Nodes []api.Node `yaml:"nodes"`
}

type jobsDocument struct {
	Jobs []api.Job `yaml:"jobs"`
}

// Load reads two scenario files and returns nodes with initialized idle capacity.
func Load(nodesPath, jobsPath string) ([]api.Node, []api.Job, error) {
	var nodeDocument nodesDocument
	if err := decode(nodesPath, &nodeDocument); err != nil {
		return nil, nil, fmt.Errorf("load nodes: %w", err)
	}
	for index := range nodeDocument.Nodes {
		node := &nodeDocument.Nodes[index]
		if err := validateResource(node.Name, "node", node.Capacity); err != nil {
			return nil, nil, err
		}
		node.Idle = api.NewResource(node.Capacity)
	}

	var jobDocument jobsDocument
	if err := decode(jobsPath, &jobDocument); err != nil {
		return nil, nil, fmt.Errorf("load jobs: %w", err)
	}
	for index := range jobDocument.Jobs {
		job := &jobDocument.Jobs[index]
		if job.Name == "" {
			return nil, nil, fmt.Errorf("job has an empty name")
		}
		if job.MinAvailable <= 0 || job.MinAvailable > job.Replicas {
			return nil, nil, fmt.Errorf("job %q has invalid minAvailable %d for %d replicas", job.Name, job.MinAvailable, job.Replicas)
		}
		if err := validateResource(job.Name, "job", job.Request); err != nil {
			return nil, nil, err
		}
		job.Allocated = api.NewResource(nil)
	}

	return nodeDocument.Nodes, jobDocument.Jobs, nil
}

func decode(path string, target any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(contents, target); err != nil {
		return err
	}
	return nil
}

func validateResource(name, kind string, resource api.Resource) error {
	if name == "" {
		return fmt.Errorf("%s has an empty name", kind)
	}
	if len(resource) == 0 {
		return fmt.Errorf("%s %q has no resources", kind, name)
	}
	for resourceName, quantity := range resource {
		if quantity <= 0 {
			return fmt.Errorf("%s %q has non-positive %s quantity %v", kind, name, resourceName, quantity)
		}
	}
	return nil
}
