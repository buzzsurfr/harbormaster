package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/buzzsurfr/harbormaster/cluster"
	"github.com/buzzsurfr/harbormaster/node"
)

var ecsSvc *ecs.ECS
var eksSvc *eks.EKS

func normalizeEcsCluster(ecsCluster *ecs.Cluster) cluster.Cluster {
	return cluster.Cluster{
		Name:      *ecsCluster.ClusterName,
		Arn:       *ecsCluster.ClusterArn,
		Scheduler: "ecs",
		Status:    *ecsCluster.Status,
	}
}

func normalizeEksCluster(eksCluster *eks.Cluster) cluster.Cluster {
	return cluster.Cluster{
		Name:      *eksCluster.Name,
		Arn:       *eksCluster.Arn,
		Scheduler: "eks",
		Status:    *eksCluster.Status,
	}
}

func normalizeEcsNode(ecsNode *ecs.ContainerInstance, c cluster.Cluster) node.Node {
	name := strings.Split(*ecsNode.ContainerInstanceArn, "/")
	return node.Node{
		Name:       name[len(name)-1],
		Arn:        *ecsNode.ContainerInstanceArn,
		InstanceID: *ecsNode.Ec2InstanceId,
		Scheduler:  "ecs",
		Status:     *ecsNode.Status,
		Cluster:    c,
	}
}

func ecsListClusters(ctx context.Context) ([]cluster.Cluster, error) {
	// ecs:ListClusters
	resultListClusters, err := ecsSvc.ListClustersWithContext(ctx, &ecs.ListClustersInput{})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				log.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				log.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				log.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}

	clusterArns := resultListClusters.ClusterArns

	// ecs:DescribeClusters
	resultDescribeClusters, err := ecsSvc.DescribeClustersWithContext(ctx, &ecs.DescribeClustersInput{
		Clusters: clusterArns,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				log.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				log.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				log.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}

	ecsClusters := resultDescribeClusters.Clusters
	clusters := make([]cluster.Cluster, len(ecsClusters))
	for i, ecsCluster := range ecsClusters {
		clusters[i] = normalizeEcsCluster(ecsCluster)
	}

	return clusters, nil
}

func eksListClusters(ctx context.Context) ([]cluster.Cluster, error) {
	// eks:ListClusters
	resultListClusters, err := eksSvc.ListClustersWithContext(ctx, &eks.ListClustersInput{})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				log.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				log.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				log.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}

	clusterNames := resultListClusters.Clusters

	// eks:DescribeClusters (per cluster)
	eksClusters := make([]eks.Cluster, len(clusterNames))
	for i, clusterName := range clusterNames {
		resultDescribeCluster, err := eksSvc.DescribeClusterWithContext(ctx, &eks.DescribeClusterInput{
			Name: clusterName,
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case ecs.ErrCodeServerException:
					log.Println(ecs.ErrCodeServerException, aerr.Error())
				case ecs.ErrCodeClientException:
					log.Println(ecs.ErrCodeClientException, aerr.Error())
				case ecs.ErrCodeInvalidParameterException:
					log.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
				default:
					log.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				log.Println(err.Error())
			}
			return nil, err
		}

		eksClusters[i] = *resultDescribeCluster.Cluster
	}

	clusters := make([]cluster.Cluster, len(eksClusters))
	for i, eksCluster := range eksClusters {
		clusters[i] = normalizeEksCluster(&eksCluster)
	}

	return clusters, nil
}

func ecsListNodes(ctx context.Context, c cluster.Cluster) ([]node.Node, error) {
	// ecs:ListContainerInstances
	resultListContainerInstances, err := ecsSvc.ListContainerInstancesWithContext(ctx, &ecs.ListContainerInstancesInput{
		Cluster: aws.String(c.Arn),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				log.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				log.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				log.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}

	containerInstanceArns := resultListContainerInstances.ContainerInstanceArns

	// ecs:DescribeContainerInstances
	resultDescribeContainerInstances, err := ecsSvc.DescribeContainerInstancesWithContext(ctx, &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(c.Arn),
		ContainerInstances: containerInstanceArns,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				log.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				log.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				log.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}

	ecsNodes := resultDescribeContainerInstances.ContainerInstances
	nodes := make([]node.Node, len(ecsNodes))
	for i, ecsNode := range ecsNodes {
		nodes[i] = normalizeEcsNode(ecsNode, c)
	}

	return nodes, nil
}

// HandleRequest is the Lambda function handler
func HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Lambda Context
	lc, _ := lambdacontext.FromContext(ctx)
	log.Print(lc.ClientContext.Client.AppPackageName)

	// Initialize ECS
	ecsSvc = ecs.New(session.Must(session.NewSession()))
	xray.AWS(ecsSvc.Client)

	// Initialize EKS
	eksSvc = eks.New(session.Must(session.NewSession()))
	xray.AWS(eksSvc.Client)

	// List ECS Clusters
	ecsClusters, _ := ecsListClusters(ctx)

	// List EKS Clusters
	eksClusters, _ := eksListClusters(ctx)

	// Merge clusters from providers
	// clusters := ecsClusters
	clusters := append(ecsClusters, eksClusters...)

	var nodes []node.Node

	for _, c := range clusters {
		switch c.Scheduler {
		case "ecs":
			clusterNodes, _ := ecsListNodes(ctx, c)
			nodes = append(nodes, clusterNodes...)
			// case "eks":
			// 	clusterNodes, _ := eksListNodes(ctx, c)
			// 	nodes = append(nodes, clusterNodes...)
		}
	}

	responseBody, _ := json.Marshal(nodes)

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
		},
	}, nil
}

func init() {
	xray.Configure(xray.Config{
		LogLevel: "trace",
	})
}

func main() {
	lambda.Start(HandleRequest)
}
