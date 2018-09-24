package cluster

// Cluster contains data for the normalized cluster
type Cluster struct {
	Name      string `locationName:"name" type:"string"`
	Arn       string `locationName:"arn" type:"string"`
	Scheduler string `locationName:"scheduler" type:"string"`
	Status    string `locationName:"status" type:"string"`
}
