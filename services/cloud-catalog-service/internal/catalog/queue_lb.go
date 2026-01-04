package catalog

import "moktopus/cloud-catalog-service/internal/model"

// AWS Queue services
var AWSQueueServices = []model.QueueConfig{
	{ID: "aws-sqs-standard", Provider: "aws", Service: "SQS", Type: "standard", MaxMessageSize: 256, RetentionDays: 14, ThroughputPerSec: 120000, PricePerMillion: 0.40, PricePerGBTransfer: 0.09, Region: "us-east-1"},
	{ID: "aws-sqs-fifo", Provider: "aws", Service: "SQS", Type: "fifo", MaxMessageSize: 256, RetentionDays: 14, ThroughputPerSec: 3000, PricePerMillion: 0.50, PricePerGBTransfer: 0.09, Region: "us-east-1"},
	{ID: "aws-sns-standard", Provider: "aws", Service: "SNS", Type: "pub-sub", MaxMessageSize: 256, RetentionDays: 0, ThroughputPerSec: 300000, PricePerMillion: 0.50, PricePerGBTransfer: 0.09, Region: "us-east-1"},
	{ID: "aws-kinesis-1shard", Provider: "aws", Service: "Kinesis", Type: "stream", MaxMessageSize: 1024, RetentionDays: 7, ThroughputPerSec: 1000, PricePerMillion: 0.014, PricePerGBTransfer: 0.013, Region: "us-east-1"},
	{ID: "aws-eventbridge", Provider: "aws", Service: "EventBridge", Type: "event-bus", MaxMessageSize: 256, RetentionDays: 1, ThroughputPerSec: 10000, PricePerMillion: 1.00, PricePerGBTransfer: 0.00, Region: "us-east-1"},
	{ID: "aws-msk-m5-large", Provider: "aws", Service: "MSK (Kafka)", Type: "stream", MaxMessageSize: 1024, RetentionDays: 7, ThroughputPerSec: 50000, PricePerMillion: 0.00, PricePerGBTransfer: 0.10, Region: "us-east-1"},
}

// GCP Queue services
var GCPQueueServices = []model.QueueConfig{
	{ID: "gcp-pubsub", Provider: "gcp", Service: "Pub/Sub", Type: "pub-sub", MaxMessageSize: 10240, RetentionDays: 7, ThroughputPerSec: 500000, PricePerMillion: 0.00, PricePerGBTransfer: 0.040, Region: "us-central1"},
	{ID: "gcp-cloud-tasks", Provider: "gcp", Service: "Cloud Tasks", Type: "task-queue", MaxMessageSize: 100, RetentionDays: 30, ThroughputPerSec: 500, PricePerMillion: 0.40, PricePerGBTransfer: 0.00, Region: "us-central1"},
	{ID: "gcp-dataflow-stream", Provider: "gcp", Service: "Dataflow", Type: "stream", MaxMessageSize: 10240, RetentionDays: 0, ThroughputPerSec: 100000, PricePerMillion: 0.00, PricePerGBTransfer: 0.018, Region: "us-central1"},
}

// Azure Queue services
var AzureQueueServices = []model.QueueConfig{
	{ID: "azure-servicebus-basic", Provider: "azure", Service: "Service Bus", Type: "queue", MaxMessageSize: 256, RetentionDays: 14, ThroughputPerSec: 100, PricePerMillion: 0.05, PricePerGBTransfer: 0.00, Region: "eastus"},
	{ID: "azure-servicebus-standard", Provider: "azure", Service: "Service Bus", Type: "pub-sub", MaxMessageSize: 256, RetentionDays: 14, ThroughputPerSec: 1000, PricePerMillion: 0.80, PricePerGBTransfer: 0.00, Region: "eastus"},
	{ID: "azure-servicebus-premium", Provider: "azure", Service: "Service Bus", Type: "pub-sub", MaxMessageSize: 1024, RetentionDays: 90, ThroughputPerSec: 10000, PricePerMillion: 0.00, PricePerGBTransfer: 0.00, Region: "eastus"},
	{ID: "azure-eventhubs-basic", Provider: "azure", Service: "Event Hubs", Type: "stream", MaxMessageSize: 256, RetentionDays: 1, ThroughputPerSec: 1000, PricePerMillion: 0.028, PricePerGBTransfer: 0.00, Region: "eastus"},
	{ID: "azure-eventhubs-standard", Provider: "azure", Service: "Event Hubs", Type: "stream", MaxMessageSize: 1024, RetentionDays: 7, ThroughputPerSec: 10000, PricePerMillion: 0.028, PricePerGBTransfer: 0.00, Region: "eastus"},
}

func AllQueueServices() []model.QueueConfig {
	all := make([]model.QueueConfig, 0, len(AWSQueueServices)+len(GCPQueueServices)+len(AzureQueueServices))
	all = append(all, AWSQueueServices...)
	all = append(all, GCPQueueServices...)
	all = append(all, AzureQueueServices...)
	return all
}

func QueueByProvider(provider string) []model.QueueConfig {
	switch provider {
	case "aws":
		return AWSQueueServices
	case "gcp":
		return GCPQueueServices
	case "azure":
		return AzureQueueServices
	default:
		return nil
	}
}

// AWS Load Balancers
var AWSLoadBalancers = []model.LoadBalancerConfig{
	{ID: "aws-alb", Provider: "aws", Service: "ALB", Type: "application", PricePerHour: 0.0225, PricePerMonth: 16.20, PricePerLCU: 0.008, Region: "us-east-1"},
	{ID: "aws-nlb", Provider: "aws", Service: "NLB", Type: "network", PricePerHour: 0.0225, PricePerMonth: 16.20, PricePerLCU: 0.006, Region: "us-east-1"},
	{ID: "aws-gwlb", Provider: "aws", Service: "GWLB", Type: "gateway", PricePerHour: 0.0125, PricePerMonth: 9.00, PricePerLCU: 0.004, Region: "us-east-1"},
}

var GCPLoadBalancers = []model.LoadBalancerConfig{
	{ID: "gcp-http-lb", Provider: "gcp", Service: "Cloud Load Balancing", Type: "application", PricePerHour: 0.025, PricePerMonth: 18.00, PricePerLCU: 0.008, Region: "us-central1"},
	{ID: "gcp-tcp-lb", Provider: "gcp", Service: "Cloud Load Balancing", Type: "network", PricePerHour: 0.025, PricePerMonth: 18.00, PricePerLCU: 0.008, Region: "us-central1"},
}

var AzureLoadBalancers = []model.LoadBalancerConfig{
	{ID: "azure-app-gw-v2", Provider: "azure", Service: "Application Gateway", Type: "application", PricePerHour: 0.246, PricePerMonth: 179.58, PricePerLCU: 0.008, Region: "eastus"},
	{ID: "azure-lb-standard", Provider: "azure", Service: "Load Balancer", Type: "network", PricePerHour: 0.025, PricePerMonth: 18.25, PricePerLCU: 0.005, Region: "eastus"},
}

func AllLoadBalancers() []model.LoadBalancerConfig {
	all := make([]model.LoadBalancerConfig, 0)
	all = append(all, AWSLoadBalancers...)
	all = append(all, GCPLoadBalancers...)
	all = append(all, AzureLoadBalancers...)
	return all
}
