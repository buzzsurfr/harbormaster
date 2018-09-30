package service

import "github.com/buzzsurfr/harbormaster/cluster"

// Node contains data for the normalized container instance/node
type Service struct {
	Name       string          `json:"name"`
	Arn        string          `json:"arn"`
	Status     string          `json:"status"`
	Cluster    cluster.Cluster `json:"cluster"`
	Scheduler  string          `json:"scheduler"`
	LaunchType string          `json:"launchType"`
	Namespace  string          `json:"namespace"`
}
