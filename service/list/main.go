package main

import (
	"context"
	"encoding/base64"
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
	"github.com/buzzsurfr/harbormaster/service"
	"github.com/kubernetes-sigs/aws-iam-authenticator/pkg/token"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

func normalizeEcsService(ecsService *ecs.Service, c cluster.Cluster) service.Service {
	return service.Service{
		Name:       *ecsService.ServiceName,
		Arn:        *ecsService.ServiceArn,
		Status:     *ecsService.Status,
		Cluster:    c,
		Scheduler:  "ecs",
		LaunchType: strings.ToLower(*ecsService.LaunchType),
		Namespace:  "",
	}
}

func normalizeEksService(eksService v1.Service, c cluster.Cluster) service.Service {
	return service.Service{
		Name:       eksService.Name,
		Arn:        "",
		Status:     eksService.Status.String(),
		Cluster:    c,
		Scheduler:  "eks",
		LaunchType: "ec2",
		Namespace:  eksService.Namespace,
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

	// return if empty
	if len(clusterArns) == 0 {
		return []cluster.Cluster{}, nil
	}

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

func eksListClusters(ctx context.Context) ([]cluster.Cluster, []eks.Cluster, error) {
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
		return nil, nil, err
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
			return nil, nil, err
		}

		eksClusters[i] = *resultDescribeCluster.Cluster
	}

	clusters := make([]cluster.Cluster, len(eksClusters))
	for i, eksCluster := range eksClusters {
		clusters[i] = normalizeEksCluster(&eksCluster)
	}

	return clusters, eksClusters, nil
}

func ecsListServices(ctx context.Context, c cluster.Cluster) ([]service.Service, error) {
	// ecs:ListServices
	resultListServices, err := ecsSvc.ListServicesWithContext(ctx, &ecs.ListServicesInput{
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

	serviceArns := resultListServices.ServiceArns

	// return if empty
	if len(serviceArns) == 0 {
		return []service.Service{}, nil
	}

	// ecs:DescribeServices
	resultDescribeServices, err := ecsSvc.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(c.Arn),
		Services: serviceArns,
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

	ecsServices := resultDescribeServices.Services
	services := make([]service.Service, len(ecsServices))
	for i, ecsService := range ecsServices {
		services[i] = normalizeEcsService(ecsService, c)
	}

	return services, nil
}

func eksListServices(ctx context.Context, c cluster.Cluster, eksCluster eks.Cluster) ([]service.Service, error) {
	// Get Kubernetes token
	gen, _ := token.NewGenerator()
	tok, _ := gen.Get(*eksCluster.Name)
	certificateAuthorityData, _ := base64.StdEncoding.DecodeString(*eksCluster.CertificateAuthority.Data)

	clientset, _ := kubernetes.NewForConfig(&rest.Config{
		Host:        *eksCluster.Endpoint,
		BearerToken: tok,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: certificateAuthorityData,
		},
	})

	eksNamespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Print(err)
	}

	eksServices := make([]v1.Service, 0)
	for _, eksNamespace := range eksNamespaces.Items {
		eksService, err := clientset.CoreV1().Services(eksNamespace.Name).List(metav1.ListOptions{})
		if err != nil {
			log.Print(err)
		}
		eksServices = append(eksServices, eksService.Items...)
	}

	services := make([]service.Service, len(eksServices))
	for i, eksService := range eksServices {
		services[i] = normalizeEksService(eksService, c)
	}

	return services, nil
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
	eksClusters, eksClustersRaw, _ := eksListClusters(ctx)

	// Merge clusters from providers
	// clusters := ecsClusters
	clusters := append(ecsClusters, eksClusters...)

	var services []service.Service

	for _, c := range clusters {
		switch c.Scheduler {
		case "ecs":
			clusterServices, _ := ecsListServices(ctx, c)
			services = append(services, clusterServices...)
		case "eks":
			var eksCluster eks.Cluster
			for i := range eksClustersRaw {
				if c.Scheduler == "eks" && c.Arn == *eksClustersRaw[i].Arn {
					eksCluster = eksClustersRaw[i]
				}
			}

			clusterServices, _ := eksListServices(ctx, c, eksCluster)
			services = append(services, clusterServices...)
		}
	}

	responseBody, _ := json.Marshal(services)

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
