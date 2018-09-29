package node

import "github.com/buzzsurfr/harbormaster/cluster"

// Node contains data for the normalized container instance/node
type Node struct {
	Name       string `json:"name"`
	Arn        string `json:"arn"`
	InstanceID string `json:"instanceId"`
	Scheduler  string `json:"scheduler"`
	Status     string `json:"status"`
	Cluster    cluster.Cluster
}
