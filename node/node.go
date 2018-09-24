package node

import "github.com/buzzsurfr/harbormaster/cluster"

// Node contains data for the normalized container instance/node
type Node struct {
	Name       string `locationName:"name" type:"string"`
	Arn        string `locationName:"arn" type:"string"`
	InstanceID string `locationName:"instanceId" type:"string"`
	Scheduler  string `locationName:"scheduler" type:"string"`
	Status     string `locationName:"status" type:"string"`
	Cluster    cluster.Cluster
}
