package cluster

// Cluster contains data for the normalized cluster
type Cluster struct {
	Name      string `json:"name"`
	Arn       string `json:"arn"`
	Scheduler string `json:"scheduler"`
	Status    string `json:"status"`
}
