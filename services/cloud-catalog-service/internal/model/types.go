package model

// Provider represents a cloud provider (AWS, GCP, Azure)
type Provider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ServiceCategory groups cloud services
type ServiceCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// Region represents a cloud provider region
type Region struct {
	ID          string  `json:"id"`
	Provider    string  `json:"provider"`
	Name        string  `json:"name"`
	DisplayName string  `json:"displayName"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// InstanceType represents a compute/cache/db instance configuration
type InstanceType struct {
	ID            string            `json:"id"`
	Provider      string            `json:"provider"`
	Service       string            `json:"service"`
	Family        string            `json:"family"`
	Name          string            `json:"name"`
	DisplayName   string            `json:"displayName"`
	VCPUs         int               `json:"vcpus"`
	MemoryGB      float64           `json:"memoryGb"`
	NetworkGbps   float64           `json:"networkGbps"`
	StorageGB     int               `json:"storageGb,omitempty"`
	PricePerHour  float64           `json:"pricePerHour"`
	PricePerMonth float64           `json:"pricePerMonth"`
	Region        string            `json:"region"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// CacheConfig represents a cache service configuration
type CacheConfig struct {
	ID               string  `json:"id"`
	Provider         string  `json:"provider"`
	Service          string  `json:"service"`
	Engine           string  `json:"engine"`
	Version          string  `json:"version"`
	NodeType         string  `json:"nodeType"`
	VCPUs            int     `json:"vcpus"`
	MemoryGB         float64 `json:"memoryGb"`
	NetworkGbps      float64 `json:"networkGbps"`
	MaxConnections   int     `json:"maxConnections"`
	PricePerHour     float64 `json:"pricePerHour"`
	PricePerMonth    float64 `json:"pricePerMonth"`
	Region           string  `json:"region"`
	ClusterSupport   bool    `json:"clusterSupport"`
	ReplicationSlots int     `json:"replicationSlots"`
}

// DatabaseConfig represents a managed database offering
type DatabaseConfig struct {
	ID            string  `json:"id"`
	Provider      string  `json:"provider"`
	Service       string  `json:"service"`
	Engine        string  `json:"engine"`
	Version       string  `json:"version"`
	InstanceClass string  `json:"instanceClass"`
	VCPUs         int     `json:"vcpus"`
	MemoryGB      float64 `json:"memoryGb"`
	StorageType   string  `json:"storageType"`
	MaxStorageGB  int     `json:"maxStorageGb"`
	IOPS          int     `json:"iops"`
	PricePerHour  float64 `json:"pricePerHour"`
	PricePerMonth float64 `json:"pricePerMonth"`
	Region        string  `json:"region"`
	MultiAZ       bool    `json:"multiAz"`
	ReadReplicas  int     `json:"readReplicas"`
}

// QueueConfig represents a managed queue/messaging offering
type QueueConfig struct {
	ID                 string  `json:"id"`
	Provider           string  `json:"provider"`
	Service            string  `json:"service"`
	Type               string  `json:"type"`
	MaxMessageSize     int     `json:"maxMessageSizeKb"`
	RetentionDays      int     `json:"retentionDays"`
	ThroughputPerSec   int     `json:"throughputPerSec"`
	PricePerMillion    float64 `json:"pricePerMillion"`
	PricePerGBTransfer float64 `json:"pricePerGbTransfer"`
	Region             string  `json:"region"`
}

// LoadBalancerConfig represents an LB offering
type LoadBalancerConfig struct {
	ID            string  `json:"id"`
	Provider      string  `json:"provider"`
	Service       string  `json:"service"`
	Type          string  `json:"type"`
	PricePerHour  float64 `json:"pricePerHour"`
	PricePerMonth float64 `json:"pricePerMonth"`
	PricePerLCU   float64 `json:"pricePerLcu"`
	Region        string  `json:"region"`
}

// CDNConfig represents a CDN offering
type CDNConfig struct {
	ID                  string  `json:"id"`
	Provider            string  `json:"provider"`
	Service             string  `json:"service"`
	PricePerGBFirst10TB float64 `json:"pricePerGbFirst10Tb"`
	PricePerGBNext40TB  float64 `json:"pricePerGbNext40Tb"`
	PricePerRequest     float64 `json:"pricePerRequest"`
	Region              string  `json:"region"`
}

// CatalogQuery filters for searching the catalog
type CatalogQuery struct {
	Provider string  `json:"provider,omitempty"`
	Service  string  `json:"service,omitempty"`
	Region   string  `json:"region,omitempty"`
	Category string  `json:"category,omitempty"`
	MinVCPUs int     `json:"minVcpus,omitempty"`
	MaxPrice float64 `json:"maxPrice,omitempty"`
}

// CatalogResponse is the unified response envelope
type CatalogResponse struct {
	Provider string      `json:"provider"`
	Service  string      `json:"service"`
	Region   string      `json:"region"`
	Count    int         `json:"count"`
	Items    interface{} `json:"items"`
}
