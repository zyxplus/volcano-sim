package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
	"github.com/zhouyingxiao/volcano-sim/pkg/loader"
	"github.com/zhouyingxiao/volcano-sim/pkg/scheduler"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("volcano-sim", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	nodesPath := flags.String("nodes", "", "path to nodes YAML")
	jobsPath := flags.String("jobs", "", "path to jobs YAML")
	queuesPath := flags.String("queues", "", "path to queues YAML")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *nodesPath == "" || *jobsPath == "" {
		return fmt.Errorf("both -nodes and -jobs are required")
	}

	nodes, jobs, err := loader.Load(*nodesPath, *jobsPath)
	if err != nil {
		return err
	}
	queues := []api.Queue{{
		Name:       "default",
		Weight:     1,
		Capability: api.ClusterTotal(nodes),
		Allocated:  api.NewResource(nil),
	}}
	if *queuesPath != "" {
		queues, err = loader.LoadQueues(*queuesPath)
		if err != nil {
			return err
		}
	}
	jobPointers := make([]*api.Job, len(jobs))
	for index := range jobs {
		jobPointers[index] = &jobs[index]
	}
	plan := scheduler.New(nodes).RunWithQueues(jobPointers, queues)
	return json.NewEncoder(output).Encode(plan)
}
