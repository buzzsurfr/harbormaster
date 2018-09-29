package cluster

// Cluster contains data for the normalized cluster
type Cluster struct {
	Name      string `json:"name" locationName:"name" type:"string"`
	Arn       string `json:"arn" locationName:"arn" type:"string"`
	Scheduler string `json:"scheduler" locationName:"scheduler" type:"string"`
	Status    string `json:"status" locationName:"status" type:"string"`
}
