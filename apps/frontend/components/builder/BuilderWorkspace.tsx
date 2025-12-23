'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from 'react';
import ReactFlow, {
  addEdge,
  Background,
  Controls,
  MiniMap,
  type Connection,
  type Edge,
  type Node,
  type ReactFlowInstance,
  useEdgesState,
  useNodesState,
} from 'reactflow';
import 'reactflow/dist/style.css';

import { createNodeData, DRAG_MIME_TYPE, NODE_KIND_GROUPS, NODE_LABELS } from './defaults';
import { useBuilderUIStore } from './store';
import { useUndoRedo } from './useUndoRedo';
import { nodeTypes } from './CustomNode';
import {
  readActiveSimulationRunId,
  readLastSimulationRun,
  subscribeActiveSimulationRunId,
  subscribeLiveSimulationSnapshot,
} from '../simulation/storage';
import type {
  BuilderNodeData,
  CacheConfig,
  CanonicalTopologyModel,
  CircuitBreakerConfig,
  DatabaseConfig,
  LoadBalancerConfig,
  NodeConfig,
  NodeKind,
  ObjectStoreConfig,
  QueueConfig,
  ServiceConfig,
  StreamConfig,
} from './types';
import type { NodeRuntimeStats, RunStateResponse } from '../simulation/types';
import { apiFetch } from '@/lib/api';

type Props = {
  projectId: string;
  /** When set, gated actions (save, validate) call this instead of hitting the API. */
  onAuthRequired?: (action: string) => void;
};

type SavedBuilderDocument = {
  nodes: Array<Node<BuilderNodeData>>;
  edges: Array<Edge>;
  updatedAt: string;
};

type TopologyGraphPayload = {
  nodes: Array<{
    id: string;
    type: string;
    config: Record<string, unknown>;
    position?: { x: number; y: number };
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
  }>;
};

type TopologyResponse = {
  projectId: string;
  graph: TopologyGraphPayload;
  ctm: TopologyGraphPayload;
  updatedAt: string;
};

const DEFAULT_NODES: Array<Node<BuilderNodeData>> = [
  {
    id: 'gateway-1',
    type: 'custom',
    position: { x: 80, y: 140 },
    data: createNodeData('api-gateway'),
  },
  {
    id: 'service-1',
    type: 'custom',
    position: { x: 380, y: 140 },
    data: createNodeData('service'),
  },
  {
    id: 'database-1',
    type: 'custom',
    position: { x: 720, y: 220 },
    data: createNodeData('database'),
  },
];

const DEFAULT_EDGES: Edge[] = [
  { id: 'e-gateway-service', source: 'gateway-1', target: 'service-1', animated: true },
  { id: 'e-service-db', source: 'service-1', target: 'database-1' },
];

function storageKey(projectId: string): string {
  return `moktopus:project:${projectId}:builder`;
}

export default function BuilderWorkspace({ projectId, onAuthRequired }: Props) {
  const wrapperRef = useRef<HTMLDivElement | null>(null);
  const rfInstanceRef = useRef<ReactFlowInstance<Node<BuilderNodeData>, Edge> | null>(null);

  const [nodes, setNodes, onNodesChange] = useNodesState<BuilderNodeData>(DEFAULT_NODES);
  const [edges, setEdges, onEdgesChange] = useEdgesState(DEFAULT_EDGES);
  const [lastSavedAt, setLastSavedAt] = useState<string | null>(null);
  const [syncState, setSyncState] = useState<string>('');
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null);
  const [nodeHeatmap, setNodeHeatmap] = useState<Record<string, NodeHeatmapStats>>({});
  const [activeRunId, setActiveRunId] = useState<string | null>(null);

  const selectedNodeId = useBuilderUIStore((state) => state.selectedNodeId);
  const setSelectedNodeId = useBuilderUIStore((state) => state.setSelectedNodeId);

  const { pushHistory, undo, redo, isProgrammatic } = useUndoRedo(setNodes, setEdges);

  const selectedNode = useMemo(() => nodes.find((node) => node.id === selectedNodeId) ?? null, [nodes, selectedNodeId]);
  const selectedEdge = useMemo(() => edges.find((edge) => edge.id === selectedEdgeId) ?? null, [edges, selectedEdgeId]);
  const heatPeak = useMemo(
    () => Math.max(1, ...Object.values(nodeHeatmap).map((stats) => stats.touchCount)),
    [nodeHeatmap],
  );
  const renderNodes = useMemo(
    () =>
      nodes.map((node) => {
        const stats = nodeHeatmap[node.id];
        if (!stats) {
          return node;
        }

        const intensity = stats.touchCount / heatPeak;
        const borderColor = colorForIntensity(intensity, stats.failureCount > 0);

        return {
          ...node,
          style: {
            ...(node.style ?? {}),
            borderWidth: 2,
            borderColor,
            boxShadow: `0 0 ${10 + Math.round(intensity * 16)}px ${fadeColor(borderColor, 0.45)}`,
          },
        };
      }),
    [heatPeak, nodeHeatmap, nodes],
  );

  const ctm: CanonicalTopologyModel = useMemo(
    () => ({
      nodes: nodes.map((node) => ({
        id: node.id,
        type: node.data.kind,
        config: node.data.config,
      })),
      edges: edges.map((edge) => ({
        id: edge.id,
        source: edge.source,
        target: edge.target,
      })),
    }),
    [nodes, edges],
  );

  const onConnect = useCallback(
    (connection: Edge | Connection) => {
      setEdges((currentEdges) => addEdge({ ...connection, animated: false }, currentEdges));
    },
    [setEdges],
  );

  const onNodeClick = useCallback(
    (_: MouseEvent, node: Node<BuilderNodeData>) => {
      setSelectedEdgeId(null);
      setSelectedNodeId(node.id);
    },
    [setSelectedNodeId],
  );

  const onEdgeClick = useCallback(
    (_: MouseEvent, edge: Edge) => {
      setSelectedNodeId(null);
      setSelectedEdgeId(edge.id);
    },
    [setSelectedNodeId],
  );

  const onPaneClick = useCallback(() => {
    setSelectedEdgeId(null);
    setSelectedNodeId(null);
  }, [setSelectedNodeId]);

  const createNode = useCallback(
    (kind: NodeKind, position: { x: number; y: number }) => {
      const nextId = `${kind}-${crypto.randomUUID().slice(0, 8)}`;
      const newNode: Node<BuilderNodeData> = {
        id: nextId,
        type: 'custom',
        position,
        data: createNodeData(kind),
      };
      setNodes((current) => [...current, newNode]);
      setSelectedEdgeId(null);
      setSelectedNodeId(nextId);
    },
    [setNodes, setSelectedNodeId],
  );

  const onDrop = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      const kind = event.dataTransfer.getData(DRAG_MIME_TYPE) as NodeKind;
      if (!kind) {
        return;
      }

      const instance = rfInstanceRef.current;
      const wrapper = wrapperRef.current;
      if (!instance || !wrapper) {
        return;
      }

      const bounds = wrapper.getBoundingClientRect();
      const position = instance.screenToFlowPosition({
        x: event.clientX - bounds.left,
        y: event.clientY - bounds.top,
      });

      createNode(kind, position);
    },
    [createNode],
  );

  const onDragOver = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'copy';
  }, []);

  const updateSelectedNode = useCallback(
    (updater: (data: BuilderNodeData) => BuilderNodeData) => {
      if (!selectedNodeId) {
        return;
      }
      setNodes((currentNodes) =>
        currentNodes.map((node) => (node.id === selectedNodeId ? { ...node, data: updater(node.data) } : node)),
      );
    },
    [selectedNodeId, setNodes],
  );

  const updateConfig = useCallback(
    (key: string, value: string | number | boolean) => {
      updateSelectedNode((currentData) => {
        const nextConfig = { ...(currentData.config as Record<string, unknown>) };
        nextConfig[key] = value;
        return {
          ...currentData,
          config: nextConfig as NodeConfig,
        };
      });
    },
    [updateSelectedNode],
  );

  const updateSelectedEdge = useCallback(
    (updater: (edge: Edge) => Edge) => {
      if (!selectedEdgeId) {
        return;
      }

      setEdges((currentEdges) =>
        currentEdges.map((edge) => (edge.id === selectedEdgeId ? updater(edge) : edge)),
      );
    },
    [selectedEdgeId, setEdges],
  );

  const updateEdgeEndpoint = useCallback(
    (field: 'source' | 'target', nodeId: string) => {
      updateSelectedEdge((edge) => {
        const nextSource = field === 'source' ? nodeId : edge.source;
        const nextTarget = field === 'target' ? nodeId : edge.target;
        if (nextSource === nextTarget) {
          setSyncState('Connection source and target must be different nodes.');
          return edge;
        }
        return { ...edge, [field]: nodeId };
      });
    },
    [updateSelectedEdge],
  );

  const saveArchitectureLocally = useCallback(() => {
    const payload: SavedBuilderDocument = {
      nodes,
      edges,
      updatedAt: new Date().toISOString(),
    };
    localStorage.setItem(storageKey(projectId), JSON.stringify(payload));
    setLastSavedAt(payload.updatedAt);
  }, [nodes, edges, projectId]);

  const saveArchitecture = useCallback(async () => {
    if (onAuthRequired) {
      onAuthRequired('save');
      return;
    }

    const graph = toGraphPayload(nodes, edges);
    setSyncState('Saving to topology-service...');

    try {
      const response = await apiFetch(`/api/v1/topology/projects/${encodeURIComponent(projectId)}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ graph }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'save failed');
      }

      const saved = await parseV1Data<TopologyResponse>(response);
      setLastSavedAt(saved.updatedAt);
      setValidationErrors([]);
      saveArchitectureLocally();
      setSyncState('Saved to topology-service and local cache.');
      return;
    } catch {
      saveArchitectureLocally();
      setSyncState('Gateway unavailable. Saved to local cache only.');
    }
  }, [nodes, edges, projectId, saveArchitectureLocally, onAuthRequired]);

  const loadArchitectureFromLocal = useCallback(() => {
    const raw = localStorage.getItem(storageKey(projectId));
    if (!raw) {
      setNodes(DEFAULT_NODES);
      setEdges(DEFAULT_EDGES);
      setSelectedEdgeId(null);
      setSelectedNodeId(null);
      setLastSavedAt(null);
      return;
    }

    try {
      const parsed = JSON.parse(raw) as SavedBuilderDocument;
      setNodes(parsed.nodes ?? []);
      setEdges(parsed.edges ?? []);
      setSelectedEdgeId(null);
      setSelectedNodeId(null);
      setLastSavedAt(parsed.updatedAt ?? null);
    } catch {
      setNodes(DEFAULT_NODES);
      setEdges(DEFAULT_EDGES);
      setSelectedEdgeId(null);
      setSelectedNodeId(null);
      setLastSavedAt(null);
    }
  }, [projectId, setEdges, setNodes, setSelectedNodeId]);

  const loadArchitecture = useCallback(async () => {
    if (onAuthRequired) {
      loadArchitectureFromLocal();
      setSyncState('Loaded from local storage.');
      return;
    }

    setSyncState('Loading topology...');
    try {
      const response = await apiFetch(`/api/v1/topology/projects/${encodeURIComponent(projectId)}`);
      if (response.status === 404) {
        loadArchitectureFromLocal();
        setSyncState('No saved topology in service. Loaded local cache.');
        return;
      }
      if (!response.ok) {
        throw new Error('unable to load topology');
      }

      const saved = await parseV1Data<TopologyResponse>(response);
      const hydrated = fromGraphPayload(saved.graph);

      // Merge with locally-saved positions to prevent position reset
      const localRaw = localStorage.getItem(storageKey(projectId));
      if (localRaw) {
        try {
          const localParsed = JSON.parse(localRaw) as SavedBuilderDocument;
          const positionMap = new Map(
            (localParsed.nodes ?? []).map((n) => [n.id, n.position])
          );
          hydrated.nodes = hydrated.nodes.map((node) => {
            const savedPos = positionMap.get(node.id);
            if (savedPos && (!node.position || (node.position.x === 0 && node.position.y === 0))) {
              return { ...node, position: savedPos };
            }
            return node;
          });
        } catch {
          // ignore parse errors
        }
      }

      setNodes(hydrated.nodes);
      setEdges(hydrated.edges);
      setSelectedEdgeId(null);
      setSelectedNodeId(null);
      setLastSavedAt(saved.updatedAt ?? null);
      setValidationErrors([]);
      setSyncState('Loaded from topology-service.');
      return;
    } catch {
      loadArchitectureFromLocal();
      setSyncState('Gateway unavailable. Loaded local cache.');
    }
  }, [loadArchitectureFromLocal, projectId, setEdges, setNodes, setSelectedNodeId, onAuthRequired]);

  const refreshHeatmap = useCallback(() => {
    const latestRun = readLastSimulationRun(projectId);
    if (!latestRun) {
      setNodeHeatmap({});
      return;
    }

    setNodeHeatmap(buildNodeHeatmap(latestRun.run.events));
  }, [projectId]);

  const applyLiveNodeStats = useCallback((stats: Record<string, NodeRuntimeStats> | undefined) => {
    if (!stats || Object.keys(stats).length === 0) {
      return;
    }
    setNodeHeatmap(buildNodeHeatmapFromRuntime(stats));
  }, []);

  const validateTopology = useCallback(async () => {
    if (onAuthRequired) {
      onAuthRequired('validate');
      return;
    }

    const graph = toGraphPayload(nodes, edges);
    setSyncState('Validating topology...');
    try {
      const response = await apiFetch(`/api/v1/topology/validate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(graph),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'validation failed');
      }

      const result = await parseV1Data<{ valid: boolean; errors: string[] }>(response);
      if (result.valid) {
        setValidationErrors([]);
        setSyncState('Topology valid.');
      } else {
        setValidationErrors(result.errors ?? []);
        setSyncState('Topology has validation issues.');
      }
    } catch {
      setSyncState('Unable to validate via service.');
    }
  }, [nodes, edges, onAuthRequired]);

  const deleteSelectedNode = useCallback(() => {
    if (!selectedNodeId) {
      return;
    }
    setNodes((currentNodes) => currentNodes.filter((node) => node.id !== selectedNodeId));
    setEdges((currentEdges) =>
      currentEdges.filter((edge) => edge.source !== selectedNodeId && edge.target !== selectedNodeId),
    );
    setSelectedEdgeId(null);
    setSelectedNodeId(null);
  }, [selectedNodeId, setEdges, setNodes, setSelectedNodeId]);

  const deleteSelectedEdge = useCallback(() => {
    if (!selectedEdgeId) {
      return;
    }
    setEdges((currentEdges) => currentEdges.filter((edge) => edge.id !== selectedEdgeId));
    setSelectedEdgeId(null);
  }, [selectedEdgeId, setEdges]);

  const deleteSelection = useCallback(() => {
    if (selectedNodeId) {
      deleteSelectedNode();
      return;
    }
    if (selectedEdgeId) {
      deleteSelectedEdge();
    }
  }, [deleteSelectedEdge, deleteSelectedNode, selectedEdgeId, selectedNodeId]);

  useEffect(() => {
    void loadArchitecture();
  }, [loadArchitecture]);

  // Auto-save positions to localStorage whenever nodes/edges change (debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      const payload: SavedBuilderDocument = {
        nodes,
        edges,
        updatedAt: new Date().toISOString(),
      };
      localStorage.setItem(storageKey(projectId), JSON.stringify(payload));
    }, 1000);
    return () => clearTimeout(timer);
  }, [nodes, edges, projectId]);

  // Push to undo/redo history when topology changes (debounced to avoid position-drag noise)
  useEffect(() => {
    if (isProgrammatic.current) return;
    const timer = setTimeout(() => {
      pushHistory(nodes, edges);
    }, 500);
    return () => clearTimeout(timer);
  }, [nodes, edges, pushHistory, isProgrammatic]);

  useEffect(() => {
    refreshHeatmap();
    setActiveRunId(readActiveSimulationRunId(projectId));

    const unsubscribe = subscribeLiveSimulationSnapshot(projectId, (snapshot) => {
      if (snapshot) {
        applyLiveNodeStats(snapshot.nodeStats);
        return;
      }
      refreshHeatmap();
    });

    const unsubscribeRunId = subscribeActiveSimulationRunId(projectId, (runId) => {
      setActiveRunId(runId);
      if (!runId) {
        refreshHeatmap();
      }
    });

    const onFocus = () => refreshHeatmap();
    window.addEventListener('focus', onFocus);

    return () => {
      unsubscribe();
      unsubscribeRunId();
      window.removeEventListener('focus', onFocus);
    };
  }, [applyLiveNodeStats, projectId, refreshHeatmap]);

  useEffect(() => {
    if (!activeRunId) {
      return;
    }

    const intervalId = window.setInterval(async () => {
      const response = await apiFetch(`/api/v1/simulation/runs/${encodeURIComponent(activeRunId)}`);
      if (!response.ok) {
        return;
      }

      const runState = await parseV1Data<RunStateResponse>(response);
      if (runState.snapshot) {
        applyLiveNodeStats(runState.snapshot.nodeStats);
      }

      if (runState.status !== 'running') {
        refreshHeatmap();
      }
    }, 500);

    return () => window.clearInterval(intervalId);
  }, [activeRunId, applyLiveNodeStats, projectId, refreshHeatmap]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        if (onAuthRequired) {
          saveArchitectureLocally();
          setSyncState('Saved locally.');
        } else {
          void saveArchitecture();
        }
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'z') {
        event.preventDefault();
        if (event.shiftKey) {
          redo();
        } else {
          undo();
        }
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'y') {
        event.preventDefault();
        redo();
      }
      if (event.key === 'Delete' || event.key === 'Backspace') {
        const target = event.target as HTMLElement;
        const tag = target.tagName.toLowerCase();
        if (tag === 'input' || tag === 'textarea' || tag === 'select' || target.isContentEditable) {
          return;
        }
        deleteSelection();
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [deleteSelection, saveArchitecture, saveArchitectureLocally, onAuthRequired, undo, redo]);

  return (
    <section className="grid h-full grid-cols-12 gap-4 p-4 overflow-hidden" aria-label="Topology Builder">
      <aside className="col-span-12 rounded-xl border bg-panel p-4 md:col-span-3 xl:col-span-2 overflow-y-auto" aria-label="Node palette">
        <h2 className="text-lg font-semibold">Node Palette</h2>
        <p className="mt-1 text-xs text-mute">Drag onto canvas to create nodes.</p>

        <div className="mt-4 space-y-3">
          {NODE_KIND_GROUPS.map((group) => (
            <details key={group.title} open={group.title === 'Generic'}>
              <summary className="mb-2 text-xs font-medium uppercase tracking-wide text-mute cursor-pointer hover:text-white select-none">
                {group.title}
              </summary>
              <div className="space-y-1.5 ml-1">
                {group.kinds.map((kind) => (
                  <div
                    key={kind}
                    className="cursor-grab rounded-md border bg-black/20 px-3 py-1.5 text-xs transition hover:border-accent hover:text-accent active:cursor-grabbing flex items-center gap-2"
                    draggable
                    onDragStart={(event) => event.dataTransfer.setData(DRAG_MIME_TYPE, kind)}
                  >
                    <span className="truncate">{NODE_LABELS[kind]}</span>
                    <span className="text-[10px] text-slate-500 font-mono">{kind}</span>
                  </div>
                ))}
              </div>
            </details>
          ))}
        </div>

        <div className="mt-6 space-y-2">
          <button
            className="w-full rounded-md bg-accent px-3 py-2 text-sm font-medium text-black"
            onClick={() => void saveArchitecture()}
            aria-label="Save architecture to server"
          >
            Save Architecture
          </button>
          <button className="w-full rounded-md border px-3 py-2 text-sm" onClick={() => void loadArchitecture()} aria-label="Reload saved topology">
            Reload Saved
          </button>
          <button className="w-full rounded-md border px-3 py-2 text-sm" onClick={() => void validateTopology()} aria-label="Validate topology structure">
            Validate Topology
          </button>
          <button className="w-full rounded-md border px-3 py-2 text-sm text-red-300" onClick={deleteSelection} aria-label="Delete selected node or edge">
            Delete Selected
          </button>
        </div>

        <p className="mt-3 text-xs text-mute">Shortcut: Ctrl/Cmd + S to save.</p>
        {syncState ? <p className="mt-2 text-xs text-mute">{syncState}</p> : null}
        {lastSavedAt ? <p className="mt-1 text-xs text-mute">Last saved: {new Date(lastSavedAt).toLocaleString()}</p> : null}
        {validationErrors.length > 0 ? (
          <ul className="mt-2 space-y-1 text-xs text-red-300">
            {validationErrors.map((error) => (
              <li key={error}>- {error}</li>
            ))}
          </ul>
        ) : null}

        <div className="mt-4 rounded-md border bg-black/20 p-3 text-xs text-mute">
          <div className="flex items-center justify-between">
            <p className="font-medium text-white">Runtime Heatmap</p>
            <button className="rounded border px-2 py-1 text-[11px]" onClick={refreshHeatmap}>
              Refresh
            </button>
          </div>
          <p className="mt-2">
            Highlights nodes based on latest simulation traffic and failures.
          </p>
          {Object.keys(nodeHeatmap).length === 0 ? (
            <p className="mt-2">No run data yet. Execute simulation to populate.</p>
          ) : (
            <div className="mt-2 space-y-1">
              {Object.entries(nodeHeatmap)
                .sort((a, b) => b[1].touchCount - a[1].touchCount)
                .slice(0, 4)
                .map(([nodeId, stats]) => (
                  <div key={nodeId} className="flex items-center justify-between">
                    <span>{nodeId}</span>
                    <span>
                      {stats.touchCount} hits{stats.failureCount ? ` | ${stats.failureCount} fails` : ''}
                    </span>
                  </div>
                ))}
            </div>
          )}
        </div>
      </aside>

      <div className="col-span-12 md:col-span-6 xl:col-span-7 flex flex-col gap-2 min-h-0">
        <div className="flex items-center justify-between rounded-lg border bg-panel px-4 py-2 shrink-0">
          <div className="flex items-center gap-4 text-xs text-mute">
            <span><strong className="text-ink-secondary">{nodes.length}</strong> nodes</span>
            <span><strong className="text-ink-secondary">{edges.length}</strong> edges</span>
            {activeRunId && <span className="text-green-400 animate-pulse">● Live simulation</span>}
          </div>
          <div className="flex items-center gap-2">
            <button
              className="rounded border px-2 py-1 text-[11px] hover:bg-white/5"
              onClick={() => rfInstanceRef.current?.fitView({ padding: 0.2 })}
            >
              Fit View
            </button>
            <button
              className="rounded border px-2 py-1 text-[11px] hover:bg-white/5"
              onClick={() => {
                const blob = new Blob([JSON.stringify({ nodes, edges }, null, 2)], { type: 'application/json' });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `topology-${projectId}.json`;
                a.click();
                URL.revokeObjectURL(url);
              }}
            >
              Export JSON
            </button>
            <button
              className="rounded border px-2 py-1 text-[11px] hover:bg-white/5"
              onClick={() => {
                if (onAuthRequired) {
                  onAuthRequired('export-iac');
                  return;
                }
                window.location.href = `/project/${projectId}/iac?mode=export`;
              }}
            >
              Export IaC
            </button>
          </div>
        </div>
        <div className="flex-1 min-h-0 overflow-hidden rounded-xl border bg-black/20" ref={wrapperRef}>
          <ReactFlow
            nodes={renderNodes}
            edges={edges}
            nodeTypes={nodeTypes}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onEdgeClick={onEdgeClick}
            onPaneClick={onPaneClick}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onInit={(instance) => {
              rfInstanceRef.current = instance;
              // Only fit view on initial load, not on every re-render
              setTimeout(() => instance.fitView({ padding: 0.15 }), 50);
            }}
            defaultEdgeOptions={{ type: 'smoothstep', style: { stroke: '#6366f1', strokeWidth: 2, opacity: 0.7 } }}
            snapToGrid
            snapGrid={[12, 12]}
            proOptions={{ hideAttribution: true }}
          >
            <Background gap={20} size={1} color="#1e293b" />
            <Controls className="!bg-slate-900/80 !border-slate-700 !shadow-lg" />
            <MiniMap
              nodeColor={(node) => {
                const kind = (node.data as BuilderNodeData)?.kind ?? '';
                if (kind.startsWith('aws-')) return '#f97316';
                if (kind.startsWith('gcp-')) return '#3b82f6';
                if (kind.startsWith('azure-')) return '#06b6d4';
                if (kind === 'database') return '#a855f7';
                if (kind === 'cache') return '#10b981';
                if (kind === 'queue' || kind === 'stream') return '#eab308';
                return '#6366f1';
              }}
              maskColor="rgba(0,0,0,0.75)"
              style={{ background: '#0f172a', borderRadius: 8 }}
              pannable
              zoomable
            />
          </ReactFlow>
        </div>
      </div>

      <aside className="col-span-12 rounded-xl border bg-panel p-4 md:col-span-3 xl:col-span-3 overflow-y-auto">
        <h2 className="text-lg font-semibold">Inspector</h2>
        {selectedNode ? (
          <div className="mt-3 space-y-3">
            <p className="text-xs uppercase tracking-wide text-mute">Editing Node</p>
            <label className="block text-xs text-mute">Label</label>
            <input
              className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
              value={selectedNode.data.label}
              onChange={(event) =>
                updateSelectedNode((data) => ({
                  ...data,
                  label: event.target.value,
                }))
              }
            />

            {/* Cloud Provider Info */}
            {(selectedNode.data.config as any)?.provider && (
              <div className="rounded-md border bg-black/30 p-3 space-y-2">
                <p className="text-xs font-medium text-accent">Cloud Provider Config</p>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <span className="text-mute">Provider</span>
                    <p className="font-medium uppercase">{(selectedNode.data.config as any).provider}</p>
                  </div>
                  <div>
                    <span className="text-mute">Region</span>
                    <p className="font-medium">{(selectedNode.data.config as any).region}</p>
                  </div>
                  <div className="col-span-2">
                    <span className="text-mute">Instance Type</span>
                    <InstanceTypeSelector
                      nodeKind={selectedNode.data.kind}
                      provider={(selectedNode.data.config as any).provider}
                      currentValue={(selectedNode.data.config as any).instanceType ?? ''}
                      onChange={(value) => updateConfig('instanceType', value)}
                    />
                  </div>
                  {(selectedNode.data.config as any).engine && (
                    <div className="col-span-2">
                      <span className="text-mute">Engine</span>
                      <p className="font-medium">{(selectedNode.data.config as any).engine}</p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {selectedNode.data.kind === 'service' ||
            selectedNode.data.kind === 'worker' ||
            selectedNode.data.kind === 'api-gateway' ||
            selectedNode.data.kind === 'stream' ||
            selectedNode.data.kind === 'aws-ecs' ||
            selectedNode.data.kind === 'aws-ec2' ||
            selectedNode.data.kind === 'aws-lambda' ||
            selectedNode.data.kind === 'gcp-cloud-run' ||
            selectedNode.data.kind === 'gcp-compute' ||
            selectedNode.data.kind === 'azure-container-apps' ||
            selectedNode.data.kind === 'azure-vm' ? (
              (() => {
                if (selectedNode.data.kind === 'stream') {
                  const config = selectedNode.data.config as StreamConfig;
                  return (
                    <>
                      <NumberField
                        label="Processing Rate"
                        value={config.processingRate}
                        onChange={(value) => updateConfig('processingRate', value)}
                        step={10}
                      />
                      <NumberField
                        label="Partitions"
                        value={config.partitions}
                        onChange={(value) => updateConfig('partitions', value)}
                        step={1}
                      />
                      <NumberField
                        label="Retention (hours)"
                        value={config.retentionHours}
                        onChange={(value) => updateConfig('retentionHours', value)}
                        step={1}
                      />
                    </>
                  );
                }

                const config = selectedNode.data.config as ServiceConfig;
                return (
                  <>
                <NumberField
                  label="CPU Limit"
                  value={config.cpuLimit}
                  onChange={(value) => updateConfig('cpuLimit', value)}
                  step={0.1}
                />
                <NumberField
                  label="Replicas"
                  value={config.replicas}
                  onChange={(value) => updateConfig('replicas', value)}
                  step={1}
                />
                <NumberField
                  label="Processing Time (ms)"
                  value={config.processingTimeMs}
                  onChange={(value) => updateConfig('processingTimeMs', value)}
                  step={1}
                />
                <NumberField
                  label="Failure Rate"
                  value={config.failureRate}
                  onChange={(value) => updateConfig('failureRate', value)}
                  step={0.001}
                />
                <CheckboxField
                  label="Autoscaling"
                  checked={config.autoscaling}
                  onChange={(checked) => updateConfig('autoscaling', checked)}
                />
                  </>
                );
              })()
            ) : null}

            {selectedNode.data.kind === 'database' ||
            selectedNode.data.kind === 'aws-rds' ||
            selectedNode.data.kind === 'gcp-cloud-sql' ||
            selectedNode.data.kind === 'azure-postgresql' ? (
              (() => {
                const config = selectedNode.data.config as DatabaseConfig;
                return (
                  <>
                <NumberField
                  label="Latency (ms)"
                  value={config.latencyMs}
                  onChange={(value) => updateConfig('latencyMs', value)}
                  step={1}
                />
                <NumberField
                  label="Connection Pool"
                  value={config.connectionPool}
                  onChange={(value) => updateConfig('connectionPool', value)}
                  step={1}
                />
                <NumberField
                  label="Failure Chance"
                  value={config.failureChance}
                  onChange={(value) => updateConfig('failureChance', value)}
                  step={0.001}
                />
                <CheckboxField
                  label="Replication"
                  checked={config.replication}
                  onChange={(checked) => updateConfig('replication', checked)}
                />
                  </>
                );
              })()
            ) : null}

            {selectedNode.data.kind === 'queue' ||
            selectedNode.data.kind === 'aws-sqs' ||
            selectedNode.data.kind === 'gcp-pubsub' ||
            selectedNode.data.kind === 'azure-service-bus' ? (
              (() => {
                const config = selectedNode.data.config as QueueConfig;
                return (
                  <>
                <NumberField
                  label="Max Depth"
                  value={config.maxDepth}
                  onChange={(value) => updateConfig('maxDepth', value)}
                  step={10}
                />
                <NumberField
                  label="Processing Rate"
                  value={config.processingRate}
                  onChange={(value) => updateConfig('processingRate', value)}
                  step={10}
                />
                <label className="block text-xs text-mute">Retry Policy</label>
                <select
                  className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                  value={config.retryPolicy}
                  onChange={(event) => updateConfig('retryPolicy', event.target.value)}
                >
                  <option className="bg-slate-900" value="none">
                    none
                  </option>
                  <option className="bg-slate-900" value="fixed">
                    fixed
                  </option>
                  <option className="bg-slate-900" value="exponential">
                    exponential
                  </option>
                </select>
                  </>
                );
              })()
            ) : null}

            {selectedNode.data.kind === 'load-balancer' ||
            selectedNode.data.kind === 'aws-alb' ||
            selectedNode.data.kind === 'gcp-lb' ||
            selectedNode.data.kind === 'azure-lb' ? (
              (() => {
                const config = selectedNode.data.config as LoadBalancerConfig;
                return (
                  <>
                <label className="block text-xs text-mute">Algorithm</label>
                <select
                  className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                  value={config.algorithm}
                  onChange={(event) => updateConfig('algorithm', event.target.value)}
                >
                  <option className="bg-slate-900" value="round-robin">
                    Round Robin
                  </option>
                  <option className="bg-slate-900" value="least-connections">
                    Least Connections
                  </option>
                </select>
                  </>
                );
              })()
            ) : null}

            {selectedNode.data.kind === 'cache' ||
            selectedNode.data.kind === 'aws-elasticache' ||
            selectedNode.data.kind === 'gcp-memorystore' ||
            selectedNode.data.kind === 'azure-cache' ? (
              (() => {
                const config = selectedNode.data.config as CacheConfig;
                return (
                  <>
                <NumberField
                  label="Latency (ms)"
                  value={config.latencyMs}
                  onChange={(value) => updateConfig('latencyMs', value)}
                  step={1}
                />
                <NumberField
                  label="Failure Chance"
                  value={config.failureChance}
                  onChange={(value) => updateConfig('failureChance', value)}
                  step={0.001}
                />
                  </>
                );
              })()
            ) : null}

            {selectedNode.data.kind === 'object-store' ||
            selectedNode.data.kind === 'aws-s3' ||
            selectedNode.data.kind === 'aws-cloudfront' ||
            selectedNode.data.kind === 'gcp-gcs' ||
            selectedNode.data.kind === 'azure-blob' ? (
              (() => {
                const config = selectedNode.data.config as ObjectStoreConfig;
                return (
                  <>
                    <NumberField
                      label="Latency (ms)"
                      value={config.latencyMs}
                      onChange={(value) => updateConfig('latencyMs', value)}
                      step={1}
                    />
                    <NumberField
                      label="Replication Factor"
                      value={config.replicationFactor}
                      onChange={(value) => updateConfig('replicationFactor', value)}
                      step={1}
                    />
                    <label className="block text-xs text-mute">Durability</label>
                    <select
                      className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                      value={config.durability}
                      onChange={(event) => updateConfig('durability', event.target.value)}
                    >
                      <option className="bg-slate-900" value="standard">
                        Standard
                      </option>
                      <option className="bg-slate-900" value="high">
                        High
                      </option>
                    </select>
                  </>
                );
              })()
            ) : null}

            {selectedNode.data.kind === 'circuit-breaker' ? (
              (() => {
                const config = selectedNode.data.config as CircuitBreakerConfig;
                return (
                  <>
                    <NumberField
                      label="Error Threshold"
                      value={config.errorThreshold}
                      onChange={(value) => updateConfig('errorThreshold', value)}
                      step={0.01}
                    />
                    <NumberField
                      label="Recovery (ms)"
                      value={config.recoveryMs}
                      onChange={(value) => updateConfig('recoveryMs', value)}
                      step={50}
                    />
                  </>
                );
              })()
            ) : null}
          </div>
        ) : selectedEdge ? (
          <div className="mt-3 space-y-3">
            <p className="text-xs uppercase tracking-wide text-mute">Editing Connection</p>
            <div>
              <label className="block text-xs text-mute">Connection ID</label>
              <input
                className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                value={selectedEdge.id}
                disabled
              />
            </div>

            <div>
              <label className="block text-xs text-mute">Source Node</label>
              <select
                className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                value={selectedEdge.source}
                onChange={(event) => updateEdgeEndpoint('source', event.target.value)}
              >
                {nodes.map((node) => (
                  <option key={node.id} className="bg-slate-900" value={node.id}>
                    {node.data.label} ({node.id})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs text-mute">Target Node</label>
              <select
                className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                value={selectedEdge.target}
                onChange={(event) => updateEdgeEndpoint('target', event.target.value)}
              >
                {nodes.map((node) => (
                  <option key={node.id} className="bg-slate-900" value={node.id}>
                    {node.data.label} ({node.id})
                  </option>
                ))}
              </select>
            </div>

            <CheckboxField
              label="Animated"
              checked={Boolean(selectedEdge.animated)}
              onChange={(checked) => updateSelectedEdge((edge) => ({ ...edge, animated: checked }))}
            />

            <button className="w-full rounded-md border px-3 py-2 text-sm text-red-300" onClick={deleteSelectedEdge}>
              Delete Connection
            </button>
          </div>
        ) : (
          <p className="mt-3 text-sm text-mute">Select a node or connection to edit.</p>
        )}

        <div className="mt-6">
          <h3 className="text-sm font-medium">CTM Preview</h3>
          <pre className="mt-2 max-h-56 overflow-auto rounded-md border bg-black/40 p-2 text-xs text-mute">
            {JSON.stringify(ctm, null, 2)}
          </pre>
        </div>
      </aside>
    </section>
  );
}

type NodeHeatmapStats = {
  touchCount: number;
  failureCount: number;
  completedCount: number;
  queuedCount: number;
};

type InstanceTypeSelectorProps = {
  nodeKind: string;
  provider: string;
  currentValue: string;
  onChange: (value: string) => void;
};

function InstanceTypeSelector({ nodeKind, provider, currentValue, onChange }: InstanceTypeSelectorProps) {
  const [options, setOptions] = useState<Array<{ name: string; displayName: string }>>([]);

  useEffect(() => {
    const catalogType = nodeKind.includes('database') || nodeKind.includes('rds') || nodeKind.includes('sql') || nodeKind.includes('postgresql')
      ? 'database'
      : nodeKind.includes('cache') || nodeKind.includes('elasticache') || nodeKind.includes('memorystore')
      ? 'cache'
      : 'compute';

    apiFetch(`/api/v1/catalog/${catalogType}?provider=${provider}`)
      .then((res) => res.json())
      .then((json) => {
        const items = json?.data?.items ?? json?.items ?? [];
        const mapped = items.map((item: any) => ({
          name: item.name ?? item.instanceClass ?? item.nodeType ?? item.id,
          displayName: item.displayName ?? item.name ?? item.instanceClass ?? item.nodeType ?? item.id,
        }));
        setOptions(mapped);
      })
      .catch(() => setOptions([]));
  }, [nodeKind, provider]);

  if (!options.length) {
    return <p className="font-medium font-mono text-xs">{currentValue || '—'}</p>;
  }

  return (
    <select
      className="w-full rounded-md border bg-transparent px-2 py-1 text-xs font-mono"
      value={currentValue}
      onChange={(e) => onChange(e.target.value)}
    >
      {!currentValue && <option value="">Select instance...</option>}
      {options.map((opt) => (
        <option key={opt.name} value={opt.name} className="bg-slate-900">
          {opt.displayName}
        </option>
      ))}
    </select>
  );
}

type NumberFieldProps = {
  label: string;
  value: number;
  onChange: (value: number) => void;
  step?: number;
};

function NumberField({ label, value, onChange, step = 1 }: NumberFieldProps) {
  return (
    <div>
      <label className="block text-xs text-mute">{label}</label>
      <input
        className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
        type="number"
        value={Number.isFinite(value) ? value : 0}
        step={step}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </div>
  );
}

type CheckboxFieldProps = {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
};

function CheckboxField({ label, checked, onChange }: CheckboxFieldProps) {
  return (
    <label className="flex items-center gap-2 text-sm">
      <input checked={checked} onChange={(event) => onChange(event.target.checked)} type="checkbox" />
      <span>{label}</span>
    </label>
  );
}

function toGraphPayload(nodes: Array<Node<BuilderNodeData>>, edges: Array<Edge>): TopologyGraphPayload {
  return {
    nodes: nodes.map((node) => ({
      id: node.id,
      type: node.data.kind,
      config: node.data.config as Record<string, unknown>,
      position: node.position,
    })),
    edges: edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
    })),
  };
}

function buildNodeHeatmap(events: Array<{ nodeId?: string; type: string }>): Record<string, NodeHeatmapStats> {
  return events.reduce<Record<string, NodeHeatmapStats>>((acc, event) => {
    if (!event.nodeId) {
      return acc;
    }

    const current = acc[event.nodeId] ?? { touchCount: 0, failureCount: 0, completedCount: 0, queuedCount: 0 };
    current.touchCount += 1;
    if (event.type === 'FAILED') {
      current.failureCount += 1;
    }
    if (event.type === 'PROCESS_COMPLETE') {
      current.completedCount += 1;
    }
    if (event.type === 'QUEUED') {
      current.queuedCount += 1;
    }
    acc[event.nodeId] = current;
    return acc;
  }, {});
}

function buildNodeHeatmapFromRuntime(stats: Record<string, NodeRuntimeStats>): Record<string, NodeHeatmapStats> {
  const result: Record<string, NodeHeatmapStats> = {};
  Object.entries(stats).forEach(([nodeID, nodeStats]) => {
    result[nodeID] = {
      touchCount: nodeStats.touchCount,
      failureCount: nodeStats.failureCount,
      completedCount: nodeStats.completedCount,
      queuedCount: nodeStats.queuedCount,
    };
  });
  return result;
}

function colorForIntensity(intensity: number, hasFailures: boolean): string {
  if (hasFailures) {
    return '#ef4444';
  }
  if (intensity > 0.66) {
    return '#f97316';
  }
  if (intensity > 0.33) {
    return '#facc15';
  }
  return '#34d399';
}

function fadeColor(color: string, alpha: number): string {
  const sanitized = color.replace('#', '');
  const r = Number.parseInt(sanitized.slice(0, 2), 16);
  const g = Number.parseInt(sanitized.slice(2, 4), 16);
  const b = Number.parseInt(sanitized.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

type V1Envelope<T> = {
  success: boolean;
  data: T;
  error: { code: string; message: string } | null;
};

async function parseV1Data<T>(response: Response): Promise<T> {
  const text = await response.text();
  if (!text || !text.trim()) {
    throw new Error(`Empty response body (status ${response.status})`);
  }
  let payload: Partial<V1Envelope<T>>;
  try {
    payload = JSON.parse(text) as Partial<V1Envelope<T>>;
  } catch {
    throw new Error(`Invalid JSON response (status ${response.status})`);
  }
  if (typeof payload !== 'object' || payload === null || typeof payload.success !== 'boolean') {
    throw new Error('Invalid v1 envelope response');
  }
  if (!payload.success) {
    throw new Error(payload.error?.message ?? `Request failed with status ${response.status}`);
  }
  if (!('data' in payload)) {
    throw new Error('Missing v1 envelope data field');
  }
  return payload.data as T;
}

function fromGraphPayload(graph: TopologyGraphPayload): {
  nodes: Array<Node<BuilderNodeData>>;
  edges: Array<Edge>;
} {
  const nodes = graph.nodes.map((node, index) => {
    const kind = (node.type as NodeKind) || 'service';
    const position = node.position ?? { x: 140 + (index % 4) * 220, y: 120 + Math.floor(index / 4) * 140 };
    return {
      id: node.id,
      type: 'custom',
      position,
      data: {
        label: NODE_LABELS[kind] ?? node.type,
        kind,
        config: node.config as NodeConfig,
      },
    };
  });

  const edges = graph.edges.map((edge) => ({
    id: edge.id,
    source: edge.source,
    target: edge.target,
    animated: false,
  }));

  return { nodes, edges };
}

function getNodeEmoji(kind: NodeKind): string {
  const map: Record<string, string> = {
    service: '⚙️', worker: '🔧', 'api-gateway': '🌐', 'load-balancer': '⚖️',
    'circuit-breaker': '🛡️', database: '🗄️', cache: '⚡', 'object-store': '📦',
    queue: '📬', stream: '🌊',
    'aws-ec2': '🖥️', 'aws-ecs': '🐳', 'aws-lambda': 'λ', 'aws-rds': '🗄️',
    'aws-elasticache': '⚡', 'aws-sqs': '📬', 'aws-alb': '⚖️', 'aws-cloudfront': '🌍', 'aws-s3': '📦',
    'gcp-compute': '🖥️', 'gcp-cloud-run': '🐳', 'gcp-cloud-sql': '🗄️',
    'gcp-memorystore': '⚡', 'gcp-pubsub': '📬', 'gcp-lb': '⚖️', 'gcp-gcs': '📦',
    'azure-vm': '🖥️', 'azure-container-apps': '🐳', 'azure-postgresql': '🗄️',
    'azure-cache': '⚡', 'azure-service-bus': '📬', 'azure-lb': '⚖️', 'azure-blob': '📦',
  };
  return map[kind] || '⚙️';
}
