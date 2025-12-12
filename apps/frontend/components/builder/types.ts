export type NodeKind =
  | 'service'
  | 'worker'
  | 'api-gateway'
  | 'load-balancer'
  | 'circuit-breaker'
  | 'database'
  | 'cache'
  | 'object-store'
  | 'queue'
  | 'stream'
  // AWS
  | 'aws-ec2'
  | 'aws-ecs'
  | 'aws-lambda'
  | 'aws-rds'
  | 'aws-elasticache'
  | 'aws-sqs'
  | 'aws-alb'
  | 'aws-cloudfront'
  | 'aws-s3'
  // GCP
  | 'gcp-compute'
  | 'gcp-cloud-run'
  | 'gcp-cloud-sql'
  | 'gcp-memorystore'
  | 'gcp-pubsub'
  | 'gcp-lb'
  | 'gcp-gcs'
  // Azure
  | 'azure-vm'
  | 'azure-container-apps'
  | 'azure-postgresql'
  | 'azure-cache'
  | 'azure-service-bus'
  | 'azure-lb'
  | 'azure-blob';

export type ServiceConfig = {
  cpuLimit: number;
  replicas: number;
  processingTimeMs: number;
  failureRate: number;
  autoscaling: boolean;
  provider?: string;
  instanceType?: string;
  region?: string;
};

export type DatabaseConfig = {
  latencyMs: number;
  connectionPool: number;
  failureChance: number;
  replication: boolean;
  provider?: string;
  instanceType?: string;
  engine?: string;
  region?: string;
};

export type QueueConfig = {
  maxDepth: number;
  processingRate: number;
  retryPolicy: 'none' | 'fixed' | 'exponential';
  provider?: string;
  instanceType?: string;
  region?: string;
};

export type StreamConfig = {
  partitions: number;
  retentionHours: number;
  processingRate: number;
};

export type LoadBalancerConfig = {
  algorithm: 'round-robin' | 'least-connections';
  provider?: string;
  instanceType?: string;
  region?: string;
};

export type CacheConfig = {
  latencyMs: number;
  failureChance: number;
  provider?: string;
  instanceType?: string;
  engine?: string;
  region?: string;
};

export type ObjectStoreConfig = {
  latencyMs: number;
  replicationFactor: number;
  durability: 'standard' | 'high';
  provider?: string;
  instanceType?: string;
  region?: string;
};

export type CircuitBreakerConfig = {
  errorThreshold: number;
  recoveryMs: number;
};

export type NodeConfig =
  | ServiceConfig
  | DatabaseConfig
  | QueueConfig
  | StreamConfig
  | LoadBalancerConfig
  | CacheConfig
  | ObjectStoreConfig
  | CircuitBreakerConfig;

export type BuilderNodeData = {
  label: string;
  kind: NodeKind;
  config: NodeConfig;
};

export type CanonicalTopologyModel = {
  nodes: Array<{
    id: string;
    type: NodeKind;
    config: NodeConfig;
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
  }>;
};
