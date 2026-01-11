package catalog

import "moktopus/cloud-catalog-service/internal/model"

var AWSCacheInstances = []model.CacheConfig{
	{ID: "aws-cache-t4g-micro", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.t4g.micro", VCPUs: 2, MemoryGB: 0.5, MaxConnections: 20000, PricePerMonth: 11.52, Region: "us-east-1", ClusterSupport: false},
	{ID: "aws-cache-t4g-small", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.t4g.small", VCPUs: 2, MemoryGB: 1.37, MaxConnections: 20000, PricePerMonth: 23.04, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-t4g-medium", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.t4g.medium", VCPUs: 2, MemoryGB: 3.09, MaxConnections: 20000, PricePerMonth: 46.08, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-r7g-large", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.r7g.large", VCPUs: 2, MemoryGB: 13.07, MaxConnections: 65000, PricePerMonth: 146.74, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-r7g-xlarge", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.r7g.xlarge", VCPUs: 4, MemoryGB: 26.32, MaxConnections: 65000, PricePerMonth: 293.47, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-r7g-2xlarge", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.r7g.2xlarge", VCPUs: 8, MemoryGB: 52.82, MaxConnections: 65000, PricePerMonth: 586.94, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-r7g-4xlarge", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.r7g.4xlarge", VCPUs: 16, MemoryGB: 105.81, MaxConnections: 65000, PricePerMonth: 1173.89, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-m7g-large", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.m7g.large", VCPUs: 2, MemoryGB: 6.38, MaxConnections: 65000, PricePerMonth: 109.66, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-m7g-xlarge", Provider: "aws", Service: "ElastiCache", Engine: "redis", NodeType: "cache.m7g.xlarge", VCPUs: 4, MemoryGB: 12.93, MaxConnections: 65000, PricePerMonth: 219.31, Region: "us-east-1", ClusterSupport: true},
	// Memcached
	{ID: "aws-cache-t4g-micro-mc", Provider: "aws", Service: "ElastiCache", Engine: "memcached", NodeType: "cache.t4g.micro", VCPUs: 2, MemoryGB: 0.5, MaxConnections: 20000, PricePerMonth: 11.52, Region: "us-east-1", ClusterSupport: true},
	{ID: "aws-cache-r7g-large-mc", Provider: "aws", Service: "ElastiCache", Engine: "memcached", NodeType: "cache.r7g.large", VCPUs: 2, MemoryGB: 13.07, MaxConnections: 65000, PricePerMonth: 146.74, Region: "us-east-1", ClusterSupport: true},
}

var GCPCacheInstances = []model.CacheConfig{
	{ID: "gcp-memorystore-basic-m1", Provider: "gcp", Service: "Memorystore", Engine: "redis", NodeType: "M1", VCPUs: 0, MemoryGB: 1, MaxConnections: 10000, PricePerMonth: 35.04, Region: "us-central1", ClusterSupport: false},
	{ID: "gcp-memorystore-basic-m2", Provider: "gcp", Service: "Memorystore", Engine: "redis", NodeType: "M2", VCPUs: 0, MemoryGB: 2, MaxConnections: 10000, PricePerMonth: 70.08, Region: "us-central1", ClusterSupport: false},
	{ID: "gcp-memorystore-basic-m3", Provider: "gcp", Service: "Memorystore", Engine: "redis", NodeType: "M3", VCPUs: 0, MemoryGB: 4, MaxConnections: 10000, PricePerMonth: 140.16, Region: "us-central1", ClusterSupport: false},
	{ID: "gcp-memorystore-basic-m4", Provider: "gcp", Service: "Memorystore", Engine: "redis", NodeType: "M4", VCPUs: 0, MemoryGB: 8, MaxConnections: 20000, PricePerMonth: 280.32, Region: "us-central1", ClusterSupport: false},
	{ID: "gcp-memorystore-standard-m5", Provider: "gcp", Service: "Memorystore", Engine: "redis", NodeType: "M5 (HA)", VCPUs: 0, MemoryGB: 16, MaxConnections: 65000, PricePerMonth: 560.64, Region: "us-central1", ClusterSupport: true},
	{ID: "gcp-memorystore-standard-m6", Provider: "gcp", Service: "Memorystore", Engine: "redis", NodeType: "M6 (HA)", VCPUs: 0, MemoryGB: 32, MaxConnections: 65000, PricePerMonth: 1121.28, Region: "us-central1", ClusterSupport: true},
}

var AzureCacheInstances = []model.CacheConfig{
	{ID: "azure-cache-c0", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "C0 Basic", VCPUs: 0, MemoryGB: 0.25, MaxConnections: 256, PricePerMonth: 16.37, Region: "eastus", ClusterSupport: false},
	{ID: "azure-cache-c1", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "C1 Basic", VCPUs: 0, MemoryGB: 1, MaxConnections: 1000, PricePerMonth: 40.92, Region: "eastus", ClusterSupport: false},
	{ID: "azure-cache-c2", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "C2 Standard", VCPUs: 0, MemoryGB: 2.5, MaxConnections: 2000, PricePerMonth: 81.76, Region: "eastus", ClusterSupport: false},
	{ID: "azure-cache-c3", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "C3 Standard", VCPUs: 2, MemoryGB: 6, MaxConnections: 7500, PricePerMonth: 163.52, Region: "eastus", ClusterSupport: false},
	{ID: "azure-cache-p1", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "P1 Premium", VCPUs: 2, MemoryGB: 6, MaxConnections: 7500, PricePerMonth: 327.04, Region: "eastus", ClusterSupport: true},
	{ID: "azure-cache-p2", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "P2 Premium", VCPUs: 4, MemoryGB: 13, MaxConnections: 15000, PricePerMonth: 654.08, Region: "eastus", ClusterSupport: true},
	{ID: "azure-cache-p3", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "P3 Premium", VCPUs: 8, MemoryGB: 26, MaxConnections: 30000, PricePerMonth: 1308.16, Region: "eastus", ClusterSupport: true},
	{ID: "azure-cache-e10", Provider: "azure", Service: "Azure Cache for Redis", Engine: "redis", NodeType: "E10 Enterprise", VCPUs: 4, MemoryGB: 12, MaxConnections: 20000, PricePerMonth: 513.00, Region: "eastus", ClusterSupport: true},
}

func AllCacheInstances() []model.CacheConfig {
	all := make([]model.CacheConfig, 0, len(AWSCacheInstances)+len(GCPCacheInstances)+len(AzureCacheInstances))
	all = append(all, AWSCacheInstances...)
	all = append(all, GCPCacheInstances...)
	all = append(all, AzureCacheInstances...)
	return all
}

func CacheByProvider(provider string) []model.CacheConfig {
	switch provider {
	case "aws":
		return AWSCacheInstances
	case "gcp":
		return GCPCacheInstances
	case "azure":
		return AzureCacheInstances
	default:
		return nil
	}
}
