package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/buzzsurfr/harbormaster/cluster"
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

	responseBody, _ := json.Marshal(clusters)

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
