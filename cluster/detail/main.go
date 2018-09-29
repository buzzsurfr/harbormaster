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

func ecsDescribeCluster(ctx context.Context, svc *ecs.ECS, name string) (cluster.Cluster, error) {
	// ecs:DescribeClusters
	resultDescribeClusters, err := svc.DescribeClustersWithContext(ctx, &ecs.DescribeClustersInput{
		Clusters: []*string{&name},
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
		return cluster.Cluster{}, err
	}

	ecsClusters := resultDescribeClusters.Clusters
	cluster := normalizeEcsCluster(ecsClusters[0])

	return cluster, nil
}

func eksDescribeCluster(ctx context.Context, svc *eks.EKS, name string) (cluster.Cluster, error) {
	// eks:DescribeCluster
	resultDescribeCluster, err := svc.DescribeClusterWithContext(ctx, &eks.DescribeClusterInput{
		Name: &name,
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
		return cluster.Cluster{}, err
	}

	eksCluster := *resultDescribeCluster.Cluster

	cluster := normalizeEksCluster(&eksCluster)

	return cluster, nil
}

// HandleRequest is the Lambda function handler
func HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Lambda Context
	lc, _ := lambdacontext.FromContext(ctx)
	log.Print(lc.ClientContext.Client.AppPackageName)

	// Determine which client to use based on scheduler
	currentScheduler := event.PathParameters["scheduler"]
	currentName := event.PathParameters["name"]

	var currentCluster cluster.Cluster

	switch currentScheduler {
	case "ecs":
		svc := ecs.New(session.Must(session.NewSession()))
		xray.AWS(svc.Client)

		currentCluster, _ = ecsDescribeCluster(ctx, svc, currentName)
	case "eks":
		svc := eks.New(session.Must(session.NewSession()))
		xray.AWS(svc.Client)

		currentCluster, _ = eksDescribeCluster(ctx, svc, currentName)
	default:
		panic("Invalid Scheduler")
	}

	responseBody, _ := json.Marshal(currentCluster)

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
		LogLevel: "info",
	})
}

func main() {
	lambda.Start(HandleRequest)
}
