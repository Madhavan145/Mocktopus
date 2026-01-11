'use client';

import { useState, useEffect, useRef } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { AppShell } from '@/components/layout/app-shell';
import { apiFetch } from '@/lib/api';
import { FileCode, Upload, Download, Copy, Check, AlertTriangle, ArrowRight, FolderOpen } from 'lucide-react';

const SAMPLE_TERRAFORM = `provider "aws" {
  region = "us-east-1"
}

resource "aws_lb" "main" {
  name               = "main-alb"
  internal           = false
  load_balancer_type = "application"
  subnets            = ["subnet-abc123", "subnet-def456"]
}

resource "aws_ecs_cluster" "main" {
  name = "production-cluster"
}

resource "aws_ecs_service" "api" {
  name            = "api-service"
  cluster         = aws_ecs_cluster.main.id
  desired_count   = 3
  launch_type     = "FARGATE"
  depends_on      = [aws_lb.main]
}

resource "aws_ecs_service" "worker" {
  name            = "worker-service"
  cluster         = aws_ecs_cluster.main.id
  desired_count   = 2
  launch_type     = "FARGATE"
  depends_on      = [aws_sqs_queue.events]
}

resource "aws_elasticache_replication_group" "cache" {
  replication_group_id = "app-cache"
  description          = "Redis cluster for session & caching"
  node_type            = "cache.r7g.large"
  num_cache_clusters   = 2
  engine               = "redis"
  engine_version       = "7.0"
}

resource "aws_db_instance" "primary" {
  identifier     = "primary-db"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = "db.r6g.large"
  multi_az       = true
  storage_type   = "gp3"
}

resource "aws_sqs_queue" "events" {
  name                       = "order-events"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 345600
}

resource "aws_sqs_queue" "notifications" {
  name = "notification-queue"
}`;

const SAMPLE_CLOUDFORMATION = `AWSTemplateFormatVersion: "2010-09-09"
Description: E-commerce microservices architecture

Resources:
  APIGateway:
    Type: AWS::ApiGatewayV2::Api
    Properties:
      Name: ecommerce-api
      ProtocolType: HTTP
      Description: Main API entry point

  ApplicationLoadBalancer:
    Type: AWS::ElasticLoadBalancingV2::LoadBalancer
    Properties:
      Name: ecommerce-alb
      Type: application
      Scheme: internet-facing

  OrderService:
    Type: AWS::ECS::Service
    DependsOn:
      - ApplicationLoadBalancer
    Properties:
      ServiceName: order-service
      DesiredCount: 3
      LaunchType: FARGATE

  PaymentService:
    Type: AWS::ECS::Service
    DependsOn:
      - OrderService
    Properties:
      ServiceName: payment-service
      DesiredCount: 2
      LaunchType: FARGATE

  OrderDatabase:
    Type: AWS::RDS::DBInstance
    Properties:
      Engine: postgres
      EngineVersion: "15.4"
      DBInstanceClass: db.r6g.large
      MultiAZ: true
      AllocatedStorage: 100

  SessionCache:
    Type: AWS::ElastiCache::ReplicationGroup
    Properties:
      ReplicationGroupDescription: Session cache
      CacheNodeType: cache.r7g.medium
      NumCacheClusters: 2
      Engine: redis

  EventQueue:
    Type: AWS::SQS::Queue
    Properties:
      QueueName: order-events
      VisibilityTimeout: 30`;

const SAMPLE_DOCKER_COMPOSE = `version: "3.8"
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - api
      - web

  api:
    build: ./services/api
    environment:
      DATABASE_URL: postgres://app:secret@postgres:5432/appdb
      REDIS_URL: redis://redis:6379
      RABBITMQ_URL: amqp://rabbitmq:5672
    depends_on:
      - postgres
      - redis
      - rabbitmq
    deploy:
      replicas: 3

  web:
    build: ./services/web
    depends_on:
      - api

  worker:
    build: ./services/worker
    environment:
      RABBITMQ_URL: amqp://rabbitmq:5672
      DATABASE_URL: postgres://app:secret@postgres:5432/appdb
    depends_on:
      - rabbitmq
      - postgres
    deploy:
      replicas: 2

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: appdb
      POSTGRES_USER: app
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    command: redis-server --maxmemory 256mb

  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "15672:15672"

volumes:
  pgdata:`;

type Format = 'terraform' | 'cloudformation' | 'docker-compose';
type Mode = 'import' | 'export';

export default function IaCPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const initialMode = searchParams.get('mode') === 'export' ? 'export' : 'import';
  const [mode, setMode] = useState<Mode>(initialMode);
  const [format, setFormat] = useState<Format>('terraform');
  const [content, setContent] = useState(SAMPLE_TERRAFORM);
  const [result, setResult] = useState<any>(null);
  const [exportResult, setExportResult] = useState<any>(null);
  const [exportFormat, setExportFormat] = useState<Format>('terraform');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);

  // Auto-load sample when format changes
  useEffect(() => {
    if (format === 'terraform') setContent(SAMPLE_TERRAFORM);
    else if (format === 'cloudformation') setContent(SAMPLE_CLOUDFORMATION);
    else setContent(SAMPLE_DOCKER_COMPOSE);
    setResult(null);
    setError('');
  }, [format]);

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (event) => {
      const text = event.target?.result as string;
      setContent(text);
      // Auto-detect format from extension
      const ext = file.name.toLowerCase();
      if (ext.endsWith('.tf') || ext.endsWith('.hcl')) setFormat('terraform');
      else if (ext.endsWith('.yaml') || ext.endsWith('.yml')) {
        if (text.includes('AWSTemplateFormatVersion') || text.includes('Resources:')) setFormat('cloudformation');
        else setFormat('docker-compose');
      } else if (ext.endsWith('.json') && text.includes('AWSTemplateFormatVersion')) setFormat('cloudformation');
    };
    reader.readAsText(file);
    e.target.value = '';
  };

  const handleImport = async () => {
    setLoading(true);
    setError('');
    setResult(null);
    try {
      const res = await apiFetch(`/api/v1/iac/import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ format, content }),
      });
      const json = await res.json();
      if (json.success) {
        setResult(json.data);
      } else {
        setError(json.error?.message || 'Import failed');
      }
    } catch (e: any) {
      setError(e.message || 'Network error');
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async () => {
    if (!result?.graph) return;
    setLoading(true);
    setError('');
    try {
      const res = await apiFetch(`/api/v1/iac/export`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          format: exportFormat,
          provider: result.provider || 'aws',
          region: result.region || 'us-east-1',
          graph: result.graph,
        }),
      });
      const json = await res.json();
      if (json.success) {
        setExportResult(json.data);
      } else {
        setError(json.error?.message || 'Export failed');
      }
    } catch (e: any) {
      setError(e.message || 'Network error');
    } finally {
      setLoading(false);
    }
  };

  const loadSample = () => {
    if (format === 'terraform') setContent(SAMPLE_TERRAFORM);
    else if (format === 'cloudformation') setContent(SAMPLE_CLOUDFORMATION);
    else setContent(SAMPLE_DOCKER_COMPOSE);
  };

  const applyToBuilder = async () => {
    if (!result?.graph) return;
    // Create a new project for this IaC import
    const projectName = `IaC Import - ${format} (${new Date().toLocaleDateString()})`;
    let newProjectId: string;
    try {
      const res = await apiFetch(`/api/v1/projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: projectName, description: `Imported from ${format} IaC configuration` }),
      });
      const json = await res.json();
      if (json.success && json.data?.id) {
        newProjectId = json.data.id;
      } else {
        // Fallback: use a generated ID
        newProjectId = `iac-${crypto.randomUUID().slice(0, 8)}`;
      }
    } catch {
      newProjectId = `iac-${crypto.randomUUID().slice(0, 8)}`;
    }

    // Store the imported graph in localStorage for the new project
    const nodes = (result.graph.nodes || []).map((node: any, index: number) => ({
      id: node.id,
      type: 'custom',
      position: { x: 140 + (index % 4) * 260, y: 100 + Math.floor(index / 4) * 160 },
      data: {
        label: node.id.replace(/-/g, ' ').replace(/\b\w/g, (c: string) => c.toUpperCase()),
        kind: node.type || 'service',
        config: node.config || {},
      },
    }));
    const edges = (result.graph.edges || []).map((edge: any) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      animated: false,
    }));
    const payload = { nodes, edges, updatedAt: new Date().toISOString() };
    localStorage.setItem(`moktopus:project:${newProjectId}:builder`, JSON.stringify(payload));
    router.push(`/project/${newProjectId}/builder`);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-heading flex items-center gap-2">
            <FileCode size={24} className="text-accent" />
            Infrastructure as Code
          </h1>
          <p className="text-body text-ink-muted mt-1">
            Import Terraform, CloudFormation, or Docker Compose into your topology — or export your architecture to IaC.
          </p>
        </div>

        {/* Mode Toggle */}
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1 rounded-lg bg-surface p-1 border">
            <button
              onClick={() => setMode('import')}
              className={`flex items-center gap-1.5 rounded-md px-4 py-2 text-sm transition-colors ${
                mode === 'import' ? 'bg-accent text-white' : 'text-ink-muted hover:text-ink'
              }`}
            >
              <Upload size={14} /> Import
            </button>
            <button
              onClick={() => setMode('export')}
              className={`flex items-center gap-1.5 rounded-md px-4 py-2 text-sm transition-colors ${
                mode === 'export' ? 'bg-accent text-white' : 'text-ink-muted hover:text-ink'
              }`}
            >
              <Download size={14} /> Export
            </button>
          </div>

          {mode === 'import' && (
            <div className="flex items-center gap-1 rounded-lg bg-surface p-1 border">
              {(['terraform', 'cloudformation', 'docker-compose'] as Format[]).map((f) => (
                <button
                  key={f}
                  onClick={() => { setFormat(f); }}
                  className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                    format === f ? 'bg-surface-elevated text-ink font-medium' : 'text-ink-muted hover:text-ink'
                  }`}
                >
                  {f === 'terraform' ? 'Terraform' : f === 'cloudformation' ? 'CloudFormation' : 'Docker Compose'}
                </button>
              ))}
            </div>
          )}
        </div>

        {mode === 'import' ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Input */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium text-ink">Input ({format})</h3>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => fileInputRef.current?.click()}
                    className="flex items-center gap-1 text-xs text-accent hover:underline"
                  >
                    <FolderOpen size={12} /> Upload file
                  </button>
                  <button onClick={loadSample} className="text-xs text-accent hover:underline">
                    Load sample
                  </button>
                </div>
              </div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".tf,.hcl,.yaml,.yml,.json"
                className="hidden"
                onChange={handleFileUpload}
              />
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                className="w-full h-[400px] rounded-xl border bg-surface p-4 font-mono text-xs text-ink placeholder:text-ink-muted focus:border-accent focus:outline-none resize-none"
                placeholder={`Paste your ${format} code here or upload a file...`}
              />
              <button
                onClick={handleImport}
                disabled={loading || !content.trim()}
                className="w-full rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-white hover:bg-accent/90 disabled:opacity-50 transition-colors"
              >
                {loading ? 'Parsing...' : 'Parse & Import'}
              </button>
            </div>

            {/* Result */}
            <div className="space-y-3">
              <h3 className="text-sm font-medium text-ink">Parsed Topology</h3>
              {error && (
                <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                  <AlertTriangle size={14} />
                  {error}
                </div>
              )}
              {result && (
                <div className="space-y-4">
                  {/* Summary */}
                  <div className="grid grid-cols-3 gap-3">
                    <div className="rounded-lg border bg-surface p-3 text-center">
                      <p className="text-2xl font-bold text-accent">{result.resourceCount}</p>
                      <p className="text-xs text-ink-muted">Resources</p>
                    </div>
                    <div className="rounded-lg border bg-surface p-3 text-center">
                      <p className="text-2xl font-bold text-ink">{result.graph?.edges?.length || 0}</p>
                      <p className="text-xs text-ink-muted">Connections</p>
                    </div>
                    <div className="rounded-lg border bg-surface p-3 text-center">
                      <p className="text-2xl font-bold text-ink">{result.provider || 'auto'}</p>
                      <p className="text-xs text-ink-muted">Provider</p>
                    </div>
                  </div>

                  {/* Warnings */}
                  {result.warnings?.length > 0 && (
                    <div className="rounded-lg border border-warning/30 bg-warning/5 p-3">
                      <p className="text-xs font-medium text-warning mb-1">Warnings</p>
                      {result.warnings.map((w: string, i: number) => (
                        <p key={i} className="text-xs text-ink-muted">• {w}</p>
                      ))}
                    </div>
                  )}

                  {/* Nodes */}
                  <div className="rounded-xl border bg-surface overflow-hidden">
                    <div className="border-b bg-surface-elevated/50 px-4 py-2">
                      <span className="text-xs font-medium text-ink-muted">Discovered Nodes ({result.graph?.nodes?.length || 0})</span>
                    </div>
                    <div className="divide-y max-h-[200px] overflow-y-auto">
                      {result.graph?.nodes?.map((node: any) => (
                        <div key={node.id} className="flex items-center justify-between px-4 py-2">
                          <div className="flex items-center gap-2">
                            <NodeIcon type={node.type} />
                            <span className="text-xs font-medium text-ink">{node.id}</span>
                          </div>
                          <span className="text-[10px] px-2 py-0.5 rounded bg-surface-elevated text-ink-muted">
                            {node.type}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Edges */}
                  {result.graph?.edges?.length > 0 && (
                    <div className="rounded-xl border bg-surface overflow-hidden">
                      <div className="border-b bg-surface-elevated/50 px-4 py-2">
                        <span className="text-xs font-medium text-ink-muted">Connections ({result.graph.edges.length})</span>
                      </div>
                      <div className="divide-y max-h-[120px] overflow-y-auto">
                        {result.graph.edges.map((edge: any) => (
                          <div key={edge.id} className="flex items-center gap-2 px-4 py-1.5 text-xs text-ink-muted">
                            <span className="font-medium text-ink">{edge.source}</span>
                            <span>→</span>
                            <span className="font-medium text-ink">{edge.target}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Actions */}
                  <div className="space-y-3">
                    <div className="flex items-center gap-2">
                      <label className="text-xs text-ink-muted">Export as:</label>
                      <select
                        value={exportFormat}
                        onChange={(e) => setExportFormat(e.target.value as Format)}
                        className="rounded-md border bg-surface px-2 py-1 text-xs text-ink focus:border-accent focus:outline-none"
                      >
                        <option value="terraform">Terraform</option>
                        <option value="cloudformation">CloudFormation</option>
                        <option value="docker-compose">Docker Compose</option>
                      </select>
                      <button
                        onClick={handleExport}
                        className="flex-1 rounded-lg border px-4 py-2 text-sm text-ink hover:bg-surface-elevated transition-colors"
                      >
                        Export
                      </button>
                    </div>
                    <button
                      onClick={applyToBuilder}
                      className="w-full rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent/90 transition-colors flex items-center justify-center gap-1.5"
                    >
                      Create Project &amp; Open in Builder <ArrowRight size={14} />
                    </button>
                  </div>
                </div>
              )}
              {!result && !error && (
                <div className="flex items-center justify-center h-[400px] rounded-xl border border-dashed text-ink-muted text-sm">
                  Paste IaC code and click Parse to see the topology
                </div>
              )}
            </div>
          </div>
        ) : (
          /* Export Mode - export current project's topology */
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <label className="text-sm text-ink-muted">Export format:</label>
              <select
                value={exportFormat}
                onChange={(e) => setExportFormat(e.target.value as Format)}
                className="rounded-md border bg-surface px-3 py-2 text-sm text-ink focus:border-accent focus:outline-none"
              >
                <option value="terraform">Terraform</option>
                <option value="cloudformation">CloudFormation</option>
                <option value="docker-compose">Docker Compose</option>
              </select>
              <button
                onClick={async () => {
                  setLoading(true); setError('');
                  try {
                    // Load current project topology from localStorage
                    const raw = localStorage.getItem(`moktopus:project:${params.id}:builder`);
                    if (!raw) { setError('No topology found. Build your architecture first.'); setLoading(false); return; }
                    const saved = JSON.parse(raw);
                    const graph = {
                      nodes: (saved.nodes || []).map((n: any) => ({ id: n.id, type: n.data?.kind || 'service', config: n.data?.config || {} })),
                      edges: (saved.edges || []).map((e: any) => ({ id: e.id, source: e.source, target: e.target })),
                    };
                    const res = await apiFetch(`/api/v1/iac/export`, {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({ format: exportFormat, provider: 'aws', region: 'us-east-1', graph }),
                    });
                    const json = await res.json();
                    if (json.success) setExportResult(json.data);
                    else setError(json.error?.message || 'Export failed');
                  } catch (e: any) { setError(e.message || 'Network error'); }
                  finally { setLoading(false); }
                }}
                className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent/90"
              >
                {loading ? 'Exporting...' : 'Export Current Topology'}
              </button>
            </div>
            {error && (
              <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                <AlertTriangle size={14} /> {error}
              </div>
            )}
            {exportResult ? (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-medium text-ink">Generated {exportFormat === 'terraform' ? 'Terraform' : exportFormat === 'cloudformation' ? 'CloudFormation' : 'Docker Compose'}</h3>
                  <button
                    onClick={() => copyToClipboard(exportResult.content)}
                    className="flex items-center gap-1.5 text-xs text-accent hover:underline"
                  >
                    {copied ? <Check size={12} /> : <Copy size={12} />}
                    {copied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <pre className="rounded-xl border bg-surface p-4 font-mono text-xs text-ink overflow-auto max-h-[500px]">
                  {exportResult.content}
                </pre>
              </div>
            ) : !error ? (
              <div className="flex flex-col items-center justify-center h-64 rounded-xl border border-dashed text-center">
                <Download size={32} className="text-ink-muted mb-3" />
                <p className="text-sm text-ink-muted">Export your current topology as Infrastructure as Code</p>
                <p className="text-xs text-ink-muted mt-1">Select a format and click Export to generate code</p>
              </div>
            ) : null}
          </div>
        )}
      </div>
    </AppShell>
  );
}

function NodeIcon({ type }: { type: string }) {
  const colors: Record<string, string> = {
    service: 'text-blue-400',
    database: 'text-purple-400',
    cache: 'text-green-400',
    queue: 'text-yellow-400',
    'load-balancer': 'text-cyan-400',
    'api-gateway': 'text-orange-400',
    cdn: 'text-pink-400',
    storage: 'text-emerald-400',
  };
  return <div className={`h-2 w-2 rounded-full bg-current ${colors[type] || 'text-ink-muted'}`} />;
}
