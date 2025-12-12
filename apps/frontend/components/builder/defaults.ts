import type { BuilderNodeData, NodeConfig, NodeKind } from './types';

export const DRAG_MIME_TYPE = 'application/x-moktopus-node-kind';

export const NODE_LABELS: Record<NodeKind, string> = {
  service: 'Service',
  worker: 'Worker',
  'api-gateway': 'API Gateway',
  'load-balancer': 'Load Balancer',
  'circuit-breaker': 'Circuit Breaker',
  database: 'Database',
  cache: 'Cache',
  'object-store': 'Object Store',
  queue: 'Queue',
  stream: 'Stream',
  // AWS
  'aws-ec2': 'EC2 Instance',
  'aws-ecs': 'ECS Fargate',
  'aws-lambda': 'Lambda Function',
  'aws-rds': 'RDS PostgreSQL',
  'aws-elasticache': 'ElastiCache Redis',
  'aws-sqs': 'SQS Queue',
  'aws-alb': 'Application LB',
  'aws-cloudfront': 'CloudFront CDN',
  'aws-s3': 'S3 Bucket',
  // GCP
  'gcp-compute': 'Compute Engine',
  'gcp-cloud-run': 'Cloud Run',
  'gcp-cloud-sql': 'Cloud SQL',
  'gcp-memorystore': 'Memorystore Redis',
  'gcp-pubsub': 'Pub/Sub',
  'gcp-lb': 'Cloud Load Balancer',
  'gcp-gcs': 'Cloud Storage',
  // Azure
  'azure-vm': 'Virtual Machine',
  'azure-container-apps': 'Container Apps',
  'azure-postgresql': 'Azure PostgreSQL',
  'azure-cache': 'Azure Cache Redis',
  'azure-service-bus': 'Service Bus',
  'azure-lb': 'Azure Load Balancer',
  'azure-blob': 'Blob Storage',
};

export const NODE_KIND_GROUPS: Array<{ title: string; kinds: NodeKind[] }> = [
  { title: 'Generic', kinds: ['service', 'worker', 'api-gateway', 'load-balancer', 'circuit-breaker', 'database', 'cache', 'object-store', 'queue', 'stream'] },
  { title: 'AWS', kinds: ['aws-ecs', 'aws-lambda', 'aws-ec2', 'aws-rds', 'aws-elasticache', 'aws-sqs', 'aws-alb', 'aws-cloudfront', 'aws-s3'] },
  { title: 'GCP', kinds: ['gcp-cloud-run', 'gcp-compute', 'gcp-cloud-sql', 'gcp-memorystore', 'gcp-pubsub', 'gcp-lb', 'gcp-gcs'] },
  { title: 'Azure', kinds: ['azure-container-apps', 'azure-vm', 'azure-postgresql', 'azure-cache', 'azure-service-bus', 'azure-lb', 'azure-blob'] },
];

export function defaultConfigFor(kind: NodeKind): NodeConfig {
  switch (kind) {
    case 'service':
    case 'worker':
    case 'api-gateway':
      return {
        cpuLimit: 1,
        replicas: 1,
        processingTimeMs: 50,
        failureRate: 0.01,
        autoscaling: false,
      };
    case 'aws-ecs':
      return {
        cpuLimit: 1,
        replicas: 2,
        processingTimeMs: 35,
        failureRate: 0.005,
        autoscaling: true,
        provider: 'aws',
        instanceType: 'FARGATE 1vCPU/2GB',
        region: 'us-east-1',
      };
    case 'aws-ec2':
      return {
        cpuLimit: 2,
        replicas: 1,
        processingTimeMs: 25,
        failureRate: 0.008,
        autoscaling: false,
        provider: 'aws',
        instanceType: 't3.medium',
        region: 'us-east-1',
      };
    case 'aws-lambda':
      return {
        cpuLimit: 0.5,
        replicas: 1,
        processingTimeMs: 100,
        failureRate: 0.002,
        autoscaling: true,
        provider: 'aws',
        instanceType: 'Lambda 512MB',
        region: 'us-east-1',
      };
    case 'gcp-cloud-run':
      return {
        cpuLimit: 1,
        replicas: 2,
        processingTimeMs: 30,
        failureRate: 0.004,
        autoscaling: true,
        provider: 'gcp',
        instanceType: 'Cloud Run 1vCPU/512MB',
        region: 'us-central1',
      };
    case 'gcp-compute':
      return {
        cpuLimit: 2,
        replicas: 1,
        processingTimeMs: 25,
        failureRate: 0.008,
        autoscaling: false,
        provider: 'gcp',
        instanceType: 'e2-medium',
        region: 'us-central1',
      };
    case 'azure-container-apps':
      return {
        cpuLimit: 1,
        replicas: 2,
        processingTimeMs: 35,
        failureRate: 0.005,
        autoscaling: true,
        provider: 'azure',
        instanceType: 'Container Apps 1vCPU/2GB',
        region: 'eastus',
      };
    case 'azure-vm':
      return {
        cpuLimit: 2,
        replicas: 1,
        processingTimeMs: 25,
        failureRate: 0.008,
        autoscaling: false,
        provider: 'azure',
        instanceType: 'Standard_D2s_v5',
        region: 'eastus',
      };
    case 'database':
      return {
        latencyMs: 12,
        connectionPool: 50,
        failureChance: 0.01,
        replication: false,
      };
    case 'aws-rds':
      return {
        latencyMs: 8,
        connectionPool: 100,
        failureChance: 0.003,
        replication: true,
        provider: 'aws',
        instanceType: 'db.r6g.large',
        engine: 'PostgreSQL 15',
        region: 'us-east-1',
      };
    case 'gcp-cloud-sql':
      return {
        latencyMs: 9,
        connectionPool: 80,
        failureChance: 0.004,
        replication: true,
        provider: 'gcp',
        instanceType: 'db-custom-2-8192',
        engine: 'PostgreSQL 15',
        region: 'us-central1',
      };
    case 'azure-postgresql':
      return {
        latencyMs: 10,
        connectionPool: 75,
        failureChance: 0.005,
        replication: true,
        provider: 'azure',
        instanceType: 'GP_Gen5_2',
        engine: 'PostgreSQL 15',
        region: 'eastus',
      };
    case 'queue':
      return {
        maxDepth: 1000,
        processingRate: 250,
        retryPolicy: 'fixed',
      };
    case 'aws-sqs':
      return {
        maxDepth: 100000,
        processingRate: 3000,
        retryPolicy: 'exponential',
        provider: 'aws',
        instanceType: 'Standard Queue',
        region: 'us-east-1',
      };
    case 'gcp-pubsub':
      return {
        maxDepth: 100000,
        processingRate: 10000,
        retryPolicy: 'exponential',
        provider: 'gcp',
        instanceType: 'Pub/Sub Topic',
        region: 'us-central1',
      };
    case 'azure-service-bus':
      return {
        maxDepth: 80000,
        processingRate: 4000,
        retryPolicy: 'exponential',
        provider: 'azure',
        instanceType: 'Premium Tier',
        region: 'eastus',
      };
    case 'load-balancer':
      return {
        algorithm: 'round-robin',
      };
    case 'aws-alb':
      return {
        algorithm: 'round-robin',
        provider: 'aws',
        instanceType: 'Application Load Balancer',
        region: 'us-east-1',
      };
    case 'gcp-lb':
      return {
        algorithm: 'round-robin',
        provider: 'gcp',
        instanceType: 'Global HTTP(S) LB',
        region: 'global',
      };
    case 'azure-lb':
      return {
        algorithm: 'round-robin',
        provider: 'azure',
        instanceType: 'Standard Load Balancer',
        region: 'eastus',
      };
    case 'circuit-breaker':
      return {
        errorThreshold: 0.08,
        recoveryMs: 1500,
      };
    case 'cache':
      return {
        latencyMs: 3,
        failureChance: 0.005,
      };
    case 'aws-elasticache':
      return {
        latencyMs: 1,
        failureChance: 0.002,
        provider: 'aws',
        instanceType: 'cache.r7g.large',
        engine: 'Redis 7.x',
        region: 'us-east-1',
      };
    case 'gcp-memorystore':
      return {
        latencyMs: 1,
        failureChance: 0.003,
        provider: 'gcp',
        instanceType: 'M1 (5GB)',
        engine: 'Redis 7.x',
        region: 'us-central1',
      };
    case 'azure-cache':
      return {
        latencyMs: 1,
        failureChance: 0.003,
        provider: 'azure',
        instanceType: 'Premium P1',
        engine: 'Redis 6.x',
        region: 'eastus',
      };
    case 'object-store':
      return {
        latencyMs: 25,
        replicationFactor: 3,
        durability: 'high',
      };
    case 'aws-s3':
      return {
        latencyMs: 20,
        replicationFactor: 3,
        durability: 'high',
        provider: 'aws',
        instanceType: 'S3 Standard',
        region: 'us-east-1',
      };
    case 'aws-cloudfront':
      return {
        latencyMs: 5,
        replicationFactor: 3,
        durability: 'high',
        provider: 'aws',
        instanceType: 'CloudFront Distribution',
        region: 'global',
      };
    case 'gcp-gcs':
      return {
        latencyMs: 22,
        replicationFactor: 3,
        durability: 'high',
        provider: 'gcp',
        instanceType: 'Standard Storage',
        region: 'us-central1',
      };
    case 'azure-blob':
      return {
        latencyMs: 24,
        replicationFactor: 3,
        durability: 'high',
        provider: 'azure',
        instanceType: 'Hot Access Tier',
        region: 'eastus',
      };
    case 'stream':
      return {
        partitions: 12,
        retentionHours: 72,
        processingRate: 1200,
      };
  }
}

export function createNodeData(kind: NodeKind): BuilderNodeData {
  return {
    kind,
    label: NODE_LABELS[kind],
    config: defaultConfigFor(kind),
  };
}
