package catalog

import "moktopus/cloud-catalog-service/internal/model"

// AWS regions
var AWSRegions = []model.Region{
	{ID: "us-east-1", Provider: "aws", Name: "us-east-1", DisplayName: "US East (N. Virginia)", Latitude: 39.0438, Longitude: -77.4874},
	{ID: "us-west-2", Provider: "aws", Name: "us-west-2", DisplayName: "US West (Oregon)", Latitude: 45.5231, Longitude: -122.6765},
	{ID: "eu-west-1", Provider: "aws", Name: "eu-west-1", DisplayName: "Europe (Ireland)", Latitude: 53.3498, Longitude: -6.2603},
	{ID: "eu-central-1", Provider: "aws", Name: "eu-central-1", DisplayName: "Europe (Frankfurt)", Latitude: 50.1109, Longitude: 8.6821},
	{ID: "ap-southeast-1", Provider: "aws", Name: "ap-southeast-1", DisplayName: "Asia Pacific (Singapore)", Latitude: 1.3521, Longitude: 103.8198},
	{ID: "ap-northeast-1", Provider: "aws", Name: "ap-northeast-1", DisplayName: "Asia Pacific (Tokyo)", Latitude: 35.6762, Longitude: 139.6503},
}

// GCP regions
var GCPRegions = []model.Region{
	{ID: "us-central1", Provider: "gcp", Name: "us-central1", DisplayName: "Iowa", Latitude: 41.2619, Longitude: -95.8608},
	{ID: "us-east1", Provider: "gcp", Name: "us-east1", DisplayName: "South Carolina", Latitude: 33.1960, Longitude: -80.0131},
	{ID: "europe-west1", Provider: "gcp", Name: "europe-west1", DisplayName: "Belgium", Latitude: 50.4478, Longitude: 3.8197},
	{ID: "europe-west3", Provider: "gcp", Name: "europe-west3", DisplayName: "Frankfurt", Latitude: 50.1109, Longitude: 8.6821},
	{ID: "asia-east1", Provider: "gcp", Name: "asia-east1", DisplayName: "Taiwan", Latitude: 24.0750, Longitude: 120.5564},
	{ID: "asia-southeast1", Provider: "gcp", Name: "asia-southeast1", DisplayName: "Singapore", Latitude: 1.3521, Longitude: 103.8198},
}

// Azure regions
var AzureRegions = []model.Region{
	{ID: "eastus", Provider: "azure", Name: "eastus", DisplayName: "East US", Latitude: 37.3719, Longitude: -79.8164},
	{ID: "westus2", Provider: "azure", Name: "westus2", DisplayName: "West US 2", Latitude: 47.2330, Longitude: -119.8527},
	{ID: "westeurope", Provider: "azure", Name: "westeurope", DisplayName: "West Europe", Latitude: 52.3667, Longitude: 4.9000},
	{ID: "northeurope", Provider: "azure", Name: "northeurope", DisplayName: "North Europe", Latitude: 53.3478, Longitude: -6.2597},
	{ID: "southeastasia", Provider: "azure", Name: "southeastasia", DisplayName: "Southeast Asia", Latitude: 1.2830, Longitude: 103.8333},
	{ID: "japaneast", Provider: "azure", Name: "japaneast", DisplayName: "Japan East", Latitude: 35.6800, Longitude: 139.7700},
}

// AllRegions returns all regions across providers
func AllRegions() []model.Region {
	all := make([]model.Region, 0, len(AWSRegions)+len(GCPRegions)+len(AzureRegions))
	all = append(all, AWSRegions...)
	all = append(all, GCPRegions...)
	all = append(all, AzureRegions...)
	return all
}

// RegionsByProvider returns regions for a specific provider
func RegionsByProvider(provider string) []model.Region {
	switch provider {
	case "aws":
		return AWSRegions
	case "gcp":
		return GCPRegions
	case "azure":
		return AzureRegions
	default:
		return nil
	}
}

// InterRegionLatencyMs provides approximate latency between regions (in ms)
var InterRegionLatencyMs = map[string]map[string]float64{
	"us-east-1": {
		"us-west-2":      62,
		"eu-west-1":      75,
		"eu-central-1":   89,
		"ap-southeast-1": 230,
		"ap-northeast-1": 160,
		"us-central1":    25,
		"us-east1":       5,
		"europe-west1":   80,
		"eastus":         2,
		"westus2":        60,
		"westeurope":     78,
	},
	"us-west-2": {
		"us-east-1":      62,
		"eu-west-1":      135,
		"eu-central-1":   148,
		"ap-southeast-1": 175,
		"ap-northeast-1": 108,
		"us-central1":    42,
		"europe-west1":   140,
		"eastus":         60,
		"westus2":        3,
		"southeastasia":  170,
	},
	"eu-west-1": {
		"us-east-1":      75,
		"us-west-2":      135,
		"eu-central-1":   15,
		"ap-southeast-1": 165,
		"ap-northeast-1": 220,
		"europe-west1":   8,
		"europe-west3":   12,
		"westeurope":     5,
		"northeurope":    3,
	},
	"ap-southeast-1": {
		"us-east-1":       230,
		"us-west-2":       175,
		"eu-west-1":       165,
		"ap-northeast-1":  68,
		"asia-east1":      35,
		"asia-southeast1": 2,
		"southeastasia":   2,
		"japaneast":       70,
	},
	"ap-northeast-1": {
		"us-east-1":      160,
		"us-west-2":      108,
		"eu-west-1":      220,
		"ap-southeast-1": 68,
		"asia-east1":     42,
		"japaneast":      3,
	},
}

// GetInterRegionLatency returns the latency between two regions
func GetInterRegionLatency(from, to string) float64 {
	if from == to {
		return 1.0 // same region, ~1ms
	}
	if latencies, ok := InterRegionLatencyMs[from]; ok {
		if lat, ok := latencies[to]; ok {
			return lat
		}
	}
	if latencies, ok := InterRegionLatencyMs[to]; ok {
		if lat, ok := latencies[from]; ok {
			return lat
		}
	}
	return 150.0 // default cross-region estimate
}
