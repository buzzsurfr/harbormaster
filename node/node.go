package node

import "github.com/buzzsurfr/harbormaster/cluster"

// Node contains data for the normalized container instance/node
type Node struct {
	Name       string `json:"name" locationName:"name" type:"string"`
	Arn        string `json:"arn" locationName:"arn" type:"string"`
	InstanceID string `json:"instanceId" locationName:"instanceId" type:"string"`
	Scheduler  string `json:"scheduler" locationName:"scheduler" type:"string"`
	Status     string `json:"status" locationName:"status" type:"string"`
	Cluster    cluster.Cluster
}
