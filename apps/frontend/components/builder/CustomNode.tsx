import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { BuilderNodeData } from './types';

const NODE_TYPE_LABELS: Record<string, string> = {
  // Generic
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
  'aws-ec2': 'EC2',
  'aws-ecs': 'ECS Fargate',
  'aws-lambda': 'Lambda',
  'aws-rds': 'RDS',
  'aws-elasticache': 'ElastiCache',
  'aws-sqs': 'SQS',
  'aws-alb': 'ALB',
  'aws-cloudfront': 'CloudFront',
  'aws-s3': 'S3',
  // GCP
  'gcp-compute': 'Compute Engine',
  'gcp-cloud-run': 'Cloud Run',
  'gcp-cloud-sql': 'Cloud SQL',
  'gcp-memorystore': 'Memorystore',
  'gcp-pubsub': 'Pub/Sub',
  'gcp-lb': 'Cloud LB',
  'gcp-gcs': 'GCS',
  // Azure
  'azure-vm': 'Azure VM',
  'azure-container-apps': 'Container Apps',
  'azure-postgresql': 'Azure PostgreSQL',
  'azure-cache': 'Azure Cache',
  'azure-service-bus': 'Service Bus',
  'azure-lb': 'Azure LB',
  'azure-blob': 'Blob Storage',
};

const NODE_COLORS: Record<string, { bg: string; border: string; text: string; glow: string }> = {
  service: { bg: 'bg-slate-900/90', border: 'border-blue-500/50', text: 'text-blue-200', glow: 'shadow-blue-500/10' },
  worker: { bg: 'bg-slate-900/90', border: 'border-indigo-500/50', text: 'text-indigo-200', glow: 'shadow-indigo-500/10' },
  'api-gateway': { bg: 'bg-slate-900/90', border: 'border-violet-500/50', text: 'text-violet-200', glow: 'shadow-violet-500/10' },
  'load-balancer': { bg: 'bg-slate-900/90', border: 'border-cyan-500/50', text: 'text-cyan-200', glow: 'shadow-cyan-500/10' },
  'circuit-breaker': { bg: 'bg-slate-900/90', border: 'border-rose-500/50', text: 'text-rose-200', glow: 'shadow-rose-500/10' },
  database: { bg: 'bg-slate-900/90', border: 'border-purple-500/50', text: 'text-purple-200', glow: 'shadow-purple-500/10' },
  cache: { bg: 'bg-slate-900/90', border: 'border-emerald-500/50', text: 'text-emerald-200', glow: 'shadow-emerald-500/10' },
  'object-store': { bg: 'bg-slate-900/90', border: 'border-amber-500/50', text: 'text-amber-200', glow: 'shadow-amber-500/10' },
  queue: { bg: 'bg-slate-900/90', border: 'border-yellow-500/50', text: 'text-yellow-200', glow: 'shadow-yellow-500/10' },
  stream: { bg: 'bg-slate-900/90', border: 'border-teal-500/50', text: 'text-teal-200', glow: 'shadow-teal-500/10' },
};

const PROVIDER_BADGES: Record<string, { label: string; color: string }> = {
  aws: { label: 'AWS', color: 'bg-orange-500/20 text-orange-300 border-orange-500/30' },
  gcp: { label: 'GCP', color: 'bg-blue-500/20 text-blue-300 border-blue-500/30' },
  azure: { label: 'Azure', color: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30' },
};

function CustomNode({ data, selected }: NodeProps<BuilderNodeData>) {
  const kind = data.kind;
  const provider = (data.config as any)?.provider as string | undefined;
  const baseKind = kind.replace(/^(aws|gcp|azure)-/, '');
  const colors = NODE_COLORS[baseKind] || NODE_COLORS[kind] || NODE_COLORS.service;
  const typeLabel = NODE_TYPE_LABELS[kind] || NODE_TYPE_LABELS[baseKind] || kind;
  const providerInfo = provider ? PROVIDER_BADGES[provider] : getProviderFromKind(kind);

  return (
    <div
      className={`
        relative rounded-xl border backdrop-blur-sm
        px-3.5 py-2.5 min-w-[170px] max-w-[220px]
        transition-all duration-200 ease-out
        ${colors.bg} ${selected ? 'border-accent ring-2 ring-accent/20 shadow-lg shadow-accent/20 scale-[1.02]' : colors.border}
        ${!selected ? `shadow-md ${colors.glow}` : ''}
        hover:shadow-lg hover:scale-[1.01]
      `}
    >
      {/* Connection handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="!w-2.5 !h-2.5 !bg-white/80 !border-2 !border-slate-600 hover:!bg-accent hover:!border-accent !transition-colors"
      />

      {/* Type badge at top */}
      <div className="flex items-center gap-1.5 mb-1.5">
        <span className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider ${colors.border} ${colors.text}`}>
          {typeLabel}
        </span>
        {providerInfo && (
          <span className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[9px] font-semibold tracking-wide ${providerInfo.color}`}>
            {providerInfo.label}
          </span>
        )}
      </div>

      {/* Node name */}
      <p className={`text-xs font-semibold truncate ${colors.text}`}>{data.label}</p>

      {/* Kind identifier */}
      <p className="text-[10px] text-slate-400 truncate font-mono mt-0.5">{kind}</p>

      {/* Status indicator dot */}
      <div className={`absolute top-2 right-2 w-1.5 h-1.5 rounded-full ${selected ? 'bg-accent' : 'bg-emerald-400/60'}`} />

      <Handle
        type="source"
        position={Position.Right}
        className="!w-2.5 !h-2.5 !bg-white/80 !border-2 !border-slate-600 hover:!bg-accent hover:!border-accent !transition-colors"
      />
    </div>
  );
}

function getProviderFromKind(kind: string): { label: string; color: string } | null {
  if (kind.startsWith('aws-')) return PROVIDER_BADGES.aws;
  if (kind.startsWith('gcp-')) return PROVIDER_BADGES.gcp;
  if (kind.startsWith('azure-')) return PROVIDER_BADGES.azure;
  return null;
}

export const nodeTypes = {
  custom: memo(CustomNode),
};

export default memo(CustomNode);
