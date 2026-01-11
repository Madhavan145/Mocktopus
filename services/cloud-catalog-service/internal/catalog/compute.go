package catalog

import "moktopus/cloud-catalog-service/internal/model"

// AWS EC2 compute instances - comprehensive list
var AWSComputeInstances = []model.InstanceType{
	// t3 family (burstable)
	{ID: "aws-t3-nano", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.nano", DisplayName: "T3 Nano", VCPUs: 2, MemoryGB: 0.5, NetworkGbps: 5, PricePerHour: 0.0052, PricePerMonth: 3.80, Region: "us-east-1"},
	{ID: "aws-t3-micro", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.micro", DisplayName: "T3 Micro", VCPUs: 2, MemoryGB: 1, NetworkGbps: 5, PricePerHour: 0.0104, PricePerMonth: 7.49, Region: "us-east-1"},
	{ID: "aws-t3-small", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.small", DisplayName: "T3 Small", VCPUs: 2, MemoryGB: 2, NetworkGbps: 5, PricePerHour: 0.0208, PricePerMonth: 14.98, Region: "us-east-1"},
	{ID: "aws-t3-medium", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.medium", DisplayName: "T3 Medium", VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, PricePerHour: 0.0416, PricePerMonth: 29.95, Region: "us-east-1"},
	{ID: "aws-t3-large", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.large", DisplayName: "T3 Large", VCPUs: 2, MemoryGB: 8, NetworkGbps: 5, PricePerHour: 0.0832, PricePerMonth: 59.90, Region: "us-east-1"},
	{ID: "aws-t3-xlarge", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.xlarge", DisplayName: "T3 XLarge", VCPUs: 4, MemoryGB: 16, NetworkGbps: 5, PricePerHour: 0.1664, PricePerMonth: 119.81, Region: "us-east-1"},
	{ID: "aws-t3-2xlarge", Provider: "aws", Service: "EC2", Family: "t3", Name: "t3.2xlarge", DisplayName: "T3 2XLarge", VCPUs: 8, MemoryGB: 32, NetworkGbps: 5, PricePerHour: 0.3328, PricePerMonth: 239.62, Region: "us-east-1"},
	// m6i family (general purpose)
	{ID: "aws-m6i-large", Provider: "aws", Service: "EC2", Family: "m6i", Name: "m6i.large", DisplayName: "M6i Large", VCPUs: 2, MemoryGB: 8, NetworkGbps: 12.5, PricePerHour: 0.096, PricePerMonth: 69.12, Region: "us-east-1"},
	{ID: "aws-m6i-xlarge", Provider: "aws", Service: "EC2", Family: "m6i", Name: "m6i.xlarge", DisplayName: "M6i XLarge", VCPUs: 4, MemoryGB: 16, NetworkGbps: 12.5, PricePerHour: 0.192, PricePerMonth: 138.24, Region: "us-east-1"},
	{ID: "aws-m6i-2xlarge", Provider: "aws", Service: "EC2", Family: "m6i", Name: "m6i.2xlarge", DisplayName: "M6i 2XLarge", VCPUs: 8, MemoryGB: 32, NetworkGbps: 12.5, PricePerHour: 0.384, PricePerMonth: 276.48, Region: "us-east-1"},
	{ID: "aws-m6i-4xlarge", Provider: "aws", Service: "EC2", Family: "m6i", Name: "m6i.4xlarge", DisplayName: "M6i 4XLarge", VCPUs: 16, MemoryGB: 64, NetworkGbps: 12.5, PricePerHour: 0.768, PricePerMonth: 552.96, Region: "us-east-1"},
	{ID: "aws-m6i-8xlarge", Provider: "aws", Service: "EC2", Family: "m6i", Name: "m6i.8xlarge", DisplayName: "M6i 8XLarge", VCPUs: 32, MemoryGB: 128, NetworkGbps: 12.5, PricePerHour: 1.536, PricePerMonth: 1105.92, Region: "us-east-1"},
	// m7g family (Graviton3)
	{ID: "aws-m7g-medium", Provider: "aws", Service: "EC2", Family: "m7g", Name: "m7g.medium", DisplayName: "M7g Medium (Graviton3)", VCPUs: 1, MemoryGB: 4, NetworkGbps: 12.5, PricePerHour: 0.0408, PricePerMonth: 29.38, Region: "us-east-1"},
	{ID: "aws-m7g-large", Provider: "aws", Service: "EC2", Family: "m7g", Name: "m7g.large", DisplayName: "M7g Large (Graviton3)", VCPUs: 2, MemoryGB: 8, NetworkGbps: 12.5, PricePerHour: 0.0816, PricePerMonth: 58.75, Region: "us-east-1"},
	{ID: "aws-m7g-xlarge", Provider: "aws", Service: "EC2", Family: "m7g", Name: "m7g.xlarge", DisplayName: "M7g XLarge (Graviton3)", VCPUs: 4, MemoryGB: 16, NetworkGbps: 12.5, PricePerHour: 0.1632, PricePerMonth: 117.50, Region: "us-east-1"},
	// c6i family (compute-optimized)
	{ID: "aws-c6i-large", Provider: "aws", Service: "EC2", Family: "c6i", Name: "c6i.large", DisplayName: "C6i Large", VCPUs: 2, MemoryGB: 4, NetworkGbps: 12.5, PricePerHour: 0.085, PricePerMonth: 61.20, Region: "us-east-1"},
	{ID: "aws-c6i-xlarge", Provider: "aws", Service: "EC2", Family: "c6i", Name: "c6i.xlarge", DisplayName: "C6i XLarge", VCPUs: 4, MemoryGB: 8, NetworkGbps: 12.5, PricePerHour: 0.170, PricePerMonth: 122.40, Region: "us-east-1"},
	{ID: "aws-c6i-2xlarge", Provider: "aws", Service: "EC2", Family: "c6i", Name: "c6i.2xlarge", DisplayName: "C6i 2XLarge", VCPUs: 8, MemoryGB: 16, NetworkGbps: 12.5, PricePerHour: 0.340, PricePerMonth: 244.80, Region: "us-east-1"},
	{ID: "aws-c6i-4xlarge", Provider: "aws", Service: "EC2", Family: "c6i", Name: "c6i.4xlarge", DisplayName: "C6i 4XLarge", VCPUs: 16, MemoryGB: 32, NetworkGbps: 12.5, PricePerHour: 0.680, PricePerMonth: 489.60, Region: "us-east-1"},
	// r6i family (memory-optimized)
	{ID: "aws-r6i-large", Provider: "aws", Service: "EC2", Family: "r6i", Name: "r6i.large", DisplayName: "R6i Large", VCPUs: 2, MemoryGB: 16, NetworkGbps: 12.5, PricePerHour: 0.126, PricePerMonth: 90.72, Region: "us-east-1"},
	{ID: "aws-r6i-xlarge", Provider: "aws", Service: "EC2", Family: "r6i", Name: "r6i.xlarge", DisplayName: "R6i XLarge", VCPUs: 4, MemoryGB: 32, NetworkGbps: 12.5, PricePerHour: 0.252, PricePerMonth: 181.44, Region: "us-east-1"},
	{ID: "aws-r6i-2xlarge", Provider: "aws", Service: "EC2", Family: "r6i", Name: "r6i.2xlarge", DisplayName: "R6i 2XLarge", VCPUs: 8, MemoryGB: 64, NetworkGbps: 12.5, PricePerHour: 0.504, PricePerMonth: 362.88, Region: "us-east-1"},
	// Fargate
	{ID: "aws-fargate-256-512", Provider: "aws", Service: "Fargate", Family: "fargate", Name: "0.25vCPU-0.5GB", DisplayName: "Fargate 0.25 vCPU", VCPUs: 0, MemoryGB: 0.5, NetworkGbps: 5, PricePerHour: 0.01262, PricePerMonth: 9.09, Region: "us-east-1"},
	{ID: "aws-fargate-1024-2048", Provider: "aws", Service: "Fargate", Family: "fargate", Name: "1vCPU-2GB", DisplayName: "Fargate 1 vCPU", VCPUs: 1, MemoryGB: 2, NetworkGbps: 5, PricePerHour: 0.05048, PricePerMonth: 36.35, Region: "us-east-1"},
	{ID: "aws-fargate-2048-4096", Provider: "aws", Service: "Fargate", Family: "fargate", Name: "2vCPU-4GB", DisplayName: "Fargate 2 vCPU", VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, PricePerHour: 0.10096, PricePerMonth: 72.69, Region: "us-east-1"},
	{ID: "aws-fargate-4096-8192", Provider: "aws", Service: "Fargate", Family: "fargate", Name: "4vCPU-8GB", DisplayName: "Fargate 4 vCPU", VCPUs: 4, MemoryGB: 8, NetworkGbps: 5, PricePerHour: 0.20192, PricePerMonth: 145.38, Region: "us-east-1"},
	// Lambda
	{ID: "aws-lambda-128", Provider: "aws", Service: "Lambda", Family: "serverless", Name: "128MB", DisplayName: "Lambda 128MB", VCPUs: 0, MemoryGB: 0.125, NetworkGbps: 0, PricePerHour: 0.0000021, PricePerMonth: 1.51, Region: "us-east-1", Attributes: map[string]string{"per_request": "true", "free_tier": "1M requests/mo"}},
	{ID: "aws-lambda-512", Provider: "aws", Service: "Lambda", Family: "serverless", Name: "512MB", DisplayName: "Lambda 512MB", VCPUs: 0, MemoryGB: 0.5, NetworkGbps: 0, PricePerHour: 0.0000083, PricePerMonth: 6.00, Region: "us-east-1", Attributes: map[string]string{"per_request": "true"}},
	{ID: "aws-lambda-1024", Provider: "aws", Service: "Lambda", Family: "serverless", Name: "1024MB", DisplayName: "Lambda 1GB", VCPUs: 1, MemoryGB: 1, NetworkGbps: 0, PricePerHour: 0.0000167, PricePerMonth: 12.00, Region: "us-east-1", Attributes: map[string]string{"per_request": "true"}},
}

// GCP Compute Engine instances
var GCPComputeInstances = []model.InstanceType{
	// e2 family (cost-optimized)
	{ID: "gcp-e2-micro", Provider: "gcp", Service: "Compute Engine", Family: "e2", Name: "e2-micro", DisplayName: "E2 Micro", VCPUs: 0, MemoryGB: 1, NetworkGbps: 1, PricePerHour: 0.00838, PricePerMonth: 6.11, Region: "us-central1"},
	{ID: "gcp-e2-small", Provider: "gcp", Service: "Compute Engine", Family: "e2", Name: "e2-small", DisplayName: "E2 Small", VCPUs: 0, MemoryGB: 2, NetworkGbps: 1, PricePerHour: 0.01675, PricePerMonth: 12.23, Region: "us-central1"},
	{ID: "gcp-e2-medium", Provider: "gcp", Service: "Compute Engine", Family: "e2", Name: "e2-medium", DisplayName: "E2 Medium", VCPUs: 1, MemoryGB: 4, NetworkGbps: 2, PricePerHour: 0.03351, PricePerMonth: 24.46, Region: "us-central1"},
	{ID: "gcp-e2-standard-2", Provider: "gcp", Service: "Compute Engine", Family: "e2", Name: "e2-standard-2", DisplayName: "E2 Standard 2", VCPUs: 2, MemoryGB: 8, NetworkGbps: 4, PricePerHour: 0.06701, PricePerMonth: 48.92, Region: "us-central1"},
	{ID: "gcp-e2-standard-4", Provider: "gcp", Service: "Compute Engine", Family: "e2", Name: "e2-standard-4", DisplayName: "E2 Standard 4", VCPUs: 4, MemoryGB: 16, NetworkGbps: 8, PricePerHour: 0.13402, PricePerMonth: 97.83, Region: "us-central1"},
	{ID: "gcp-e2-standard-8", Provider: "gcp", Service: "Compute Engine", Family: "e2", Name: "e2-standard-8", DisplayName: "E2 Standard 8", VCPUs: 8, MemoryGB: 32, NetworkGbps: 16, PricePerHour: 0.26805, PricePerMonth: 195.66, Region: "us-central1"},
	// n2 family (general purpose)
	{ID: "gcp-n2-standard-2", Provider: "gcp", Service: "Compute Engine", Family: "n2", Name: "n2-standard-2", DisplayName: "N2 Standard 2", VCPUs: 2, MemoryGB: 8, NetworkGbps: 10, PricePerHour: 0.0971, PricePerMonth: 70.91, Region: "us-central1"},
	{ID: "gcp-n2-standard-4", Provider: "gcp", Service: "Compute Engine", Family: "n2", Name: "n2-standard-4", DisplayName: "N2 Standard 4", VCPUs: 4, MemoryGB: 16, NetworkGbps: 10, PricePerHour: 0.1942, PricePerMonth: 141.82, Region: "us-central1"},
	{ID: "gcp-n2-standard-8", Provider: "gcp", Service: "Compute Engine", Family: "n2", Name: "n2-standard-8", DisplayName: "N2 Standard 8", VCPUs: 8, MemoryGB: 32, NetworkGbps: 16, PricePerHour: 0.3884, PricePerMonth: 283.64, Region: "us-central1"},
	{ID: "gcp-n2-standard-16", Provider: "gcp", Service: "Compute Engine", Family: "n2", Name: "n2-standard-16", DisplayName: "N2 Standard 16", VCPUs: 16, MemoryGB: 64, NetworkGbps: 32, PricePerHour: 0.7769, PricePerMonth: 567.28, Region: "us-central1"},
	// c3 family (compute-optimized)
	{ID: "gcp-c3-standard-4", Provider: "gcp", Service: "Compute Engine", Family: "c3", Name: "c3-standard-4", DisplayName: "C3 Standard 4", VCPUs: 4, MemoryGB: 16, NetworkGbps: 23, PricePerHour: 0.2094, PricePerMonth: 152.86, Region: "us-central1"},
	{ID: "gcp-c3-standard-8", Provider: "gcp", Service: "Compute Engine", Family: "c3", Name: "c3-standard-8", DisplayName: "C3 Standard 8", VCPUs: 8, MemoryGB: 32, NetworkGbps: 23, PricePerHour: 0.4188, PricePerMonth: 305.72, Region: "us-central1"},
	// Cloud Run
	{ID: "gcp-cloudrun-1cpu-512mb", Provider: "gcp", Service: "Cloud Run", Family: "serverless", Name: "1cpu-512mb", DisplayName: "Cloud Run 1 CPU", VCPUs: 1, MemoryGB: 0.5, NetworkGbps: 1, PricePerHour: 0.00002400, PricePerMonth: 17.28, Region: "us-central1", Attributes: map[string]string{"per_request": "true"}},
	{ID: "gcp-cloudrun-2cpu-1gb", Provider: "gcp", Service: "Cloud Run", Family: "serverless", Name: "2cpu-1gb", DisplayName: "Cloud Run 2 CPU", VCPUs: 2, MemoryGB: 1, NetworkGbps: 2, PricePerHour: 0.00004800, PricePerMonth: 34.56, Region: "us-central1", Attributes: map[string]string{"per_request": "true"}},
	{ID: "gcp-cloudrun-4cpu-4gb", Provider: "gcp", Service: "Cloud Run", Family: "serverless", Name: "4cpu-4gb", DisplayName: "Cloud Run 4 CPU", VCPUs: 4, MemoryGB: 4, NetworkGbps: 4, PricePerHour: 0.00009600, PricePerMonth: 69.12, Region: "us-central1", Attributes: map[string]string{"per_request": "true"}},
	// GKE Autopilot
	{ID: "gcp-gke-autopilot-small", Provider: "gcp", Service: "GKE Autopilot", Family: "kubernetes", Name: "0.5cpu-512mb", DisplayName: "GKE Autopilot 0.5 CPU", VCPUs: 0, MemoryGB: 0.5, NetworkGbps: 1, PricePerHour: 0.0325, PricePerMonth: 23.72, Region: "us-central1"},
}

// Azure compute instances
var AzureComputeInstances = []model.InstanceType{
	// B series (burstable)
	{ID: "azure-b1s", Provider: "azure", Service: "Virtual Machines", Family: "B", Name: "B1s", DisplayName: "B1s", VCPUs: 1, MemoryGB: 1, NetworkGbps: 0.32, PricePerHour: 0.0104, PricePerMonth: 7.59, Region: "eastus"},
	{ID: "azure-b2s", Provider: "azure", Service: "Virtual Machines", Family: "B", Name: "B2s", DisplayName: "B2s", VCPUs: 2, MemoryGB: 4, NetworkGbps: 0.8, PricePerHour: 0.0416, PricePerMonth: 30.37, Region: "eastus"},
	{ID: "azure-b4ms", Provider: "azure", Service: "Virtual Machines", Family: "B", Name: "B4ms", DisplayName: "B4ms", VCPUs: 4, MemoryGB: 16, NetworkGbps: 3.2, PricePerHour: 0.166, PricePerMonth: 121.18, Region: "eastus"},
	{ID: "azure-b8ms", Provider: "azure", Service: "Virtual Machines", Family: "B", Name: "B8ms", DisplayName: "B8ms", VCPUs: 8, MemoryGB: 32, NetworkGbps: 6.4, PricePerHour: 0.333, PricePerMonth: 243.09, Region: "eastus"},
	// D series (general purpose)
	{ID: "azure-d2s-v5", Provider: "azure", Service: "Virtual Machines", Family: "Dsv5", Name: "D2s_v5", DisplayName: "D2s v5", VCPUs: 2, MemoryGB: 8, NetworkGbps: 3.75, PricePerHour: 0.096, PricePerMonth: 70.08, Region: "eastus"},
	{ID: "azure-d4s-v5", Provider: "azure", Service: "Virtual Machines", Family: "Dsv5", Name: "D4s_v5", DisplayName: "D4s v5", VCPUs: 4, MemoryGB: 16, NetworkGbps: 7.5, PricePerHour: 0.192, PricePerMonth: 140.16, Region: "eastus"},
	{ID: "azure-d8s-v5", Provider: "azure", Service: "Virtual Machines", Family: "Dsv5", Name: "D8s_v5", DisplayName: "D8s v5", VCPUs: 8, MemoryGB: 32, NetworkGbps: 12.5, PricePerHour: 0.384, PricePerMonth: 280.32, Region: "eastus"},
	{ID: "azure-d16s-v5", Provider: "azure", Service: "Virtual Machines", Family: "Dsv5", Name: "D16s_v5", DisplayName: "D16s v5", VCPUs: 16, MemoryGB: 64, NetworkGbps: 12.5, PricePerHour: 0.768, PricePerMonth: 560.64, Region: "eastus"},
	// E series (memory-optimized)
	{ID: "azure-e2s-v5", Provider: "azure", Service: "Virtual Machines", Family: "Esv5", Name: "E2s_v5", DisplayName: "E2s v5", VCPUs: 2, MemoryGB: 16, NetworkGbps: 3.75, PricePerHour: 0.126, PricePerMonth: 91.98, Region: "eastus"},
	{ID: "azure-e4s-v5", Provider: "azure", Service: "Virtual Machines", Family: "Esv5", Name: "E4s_v5", DisplayName: "E4s v5", VCPUs: 4, MemoryGB: 32, NetworkGbps: 7.5, PricePerHour: 0.252, PricePerMonth: 183.96, Region: "eastus"},
	// F series (compute-optimized)
	{ID: "azure-f2s-v2", Provider: "azure", Service: "Virtual Machines", Family: "Fsv2", Name: "F2s_v2", DisplayName: "F2s v2", VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, PricePerHour: 0.0846, PricePerMonth: 61.76, Region: "eastus"},
	{ID: "azure-f4s-v2", Provider: "azure", Service: "Virtual Machines", Family: "Fsv2", Name: "F4s_v2", DisplayName: "F4s v2", VCPUs: 4, MemoryGB: 8, NetworkGbps: 7.5, PricePerHour: 0.169, PricePerMonth: 123.37, Region: "eastus"},
	// Container Apps
	{ID: "azure-container-apps-05", Provider: "azure", Service: "Container Apps", Family: "serverless", Name: "0.5cpu-1gb", DisplayName: "Container Apps 0.5 vCPU", VCPUs: 0, MemoryGB: 1, NetworkGbps: 5, PricePerHour: 0.024, PricePerMonth: 17.52, Region: "eastus", Attributes: map[string]string{"per_request": "true"}},
	{ID: "azure-container-apps-1", Provider: "azure", Service: "Container Apps", Family: "serverless", Name: "1cpu-2gb", DisplayName: "Container Apps 1 vCPU", VCPUs: 1, MemoryGB: 2, NetworkGbps: 5, PricePerHour: 0.048, PricePerMonth: 35.04, Region: "eastus", Attributes: map[string]string{"per_request": "true"}},
	{ID: "azure-container-apps-2", Provider: "azure", Service: "Container Apps", Family: "serverless", Name: "2cpu-4gb", DisplayName: "Container Apps 2 vCPU", VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, PricePerHour: 0.096, PricePerMonth: 70.08, Region: "eastus", Attributes: map[string]string{"per_request": "true"}},
	// AKS
	{ID: "azure-aks-d2s", Provider: "azure", Service: "AKS", Family: "kubernetes", Name: "D2s_v5 node", DisplayName: "AKS D2s v5 Node", VCPUs: 2, MemoryGB: 8, NetworkGbps: 3.75, PricePerHour: 0.096, PricePerMonth: 70.08, Region: "eastus"},
	// Azure Functions
	{ID: "azure-functions-consumption", Provider: "azure", Service: "Azure Functions", Family: "serverless", Name: "Consumption", DisplayName: "Functions Consumption", VCPUs: 0, MemoryGB: 1.5, NetworkGbps: 0, PricePerHour: 0.000016, PricePerMonth: 0.20, Region: "eastus", Attributes: map[string]string{"per_request": "true", "free_tier": "1M executions/mo"}},
}

// AllComputeInstances returns all compute instances
func AllComputeInstances() []model.InstanceType {
	all := make([]model.InstanceType, 0, len(AWSComputeInstances)+len(GCPComputeInstances)+len(AzureComputeInstances))
	all = append(all, AWSComputeInstances...)
	all = append(all, GCPComputeInstances...)
	all = append(all, AzureComputeInstances...)
	return all
}

func ComputeByProvider(provider string) []model.InstanceType {
	switch provider {
	case "aws":
		return AWSComputeInstances
	case "gcp":
		return GCPComputeInstances
	case "azure":
		return AzureComputeInstances
	default:
		return nil
	}
}
