'use client';

import { useCallback, useMemo, useRef, useState, type MouseEvent } from 'react';
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
import { nodeTypes } from './CustomNode';
import type { BuilderNodeData, NodeKind } from './types';

const PLAYGROUND_STORAGE_KEY = 'moktopus.playground.topology';

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

/**
 * PlaygroundBuilder — anonymous-accessible topology canvas.
 * Full drag-and-drop, no authentication required.
 * State is persisted to localStorage so it survives page reloads.
 */
export default function PlaygroundBuilder() {
  const wrapperRef = useRef<HTMLDivElement | null>(null);
  const rfInstanceRef = useRef<ReactFlowInstance<Node<BuilderNodeData>, Edge> | null>(null);

  // Load initial state from localStorage
  const initialState = useMemo(() => {
    if (typeof window === 'undefined') return { nodes: DEFAULT_NODES, edges: DEFAULT_EDGES };
    try {
      const raw = localStorage.getItem(PLAYGROUND_STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        return { nodes: parsed.nodes ?? DEFAULT_NODES, edges: parsed.edges ?? DEFAULT_EDGES };
      }
    } catch { /* ignore */ }
    return { nodes: DEFAULT_NODES, edges: DEFAULT_EDGES };
  }, []);

  const [nodes, setNodes, onNodesChange] = useNodesState<BuilderNodeData>(initialState.nodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialState.edges);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const selectedNode = useMemo(() => nodes.find((n) => n.id === selectedNodeId) ?? null, [nodes, selectedNodeId]);

  // Persist to localStorage on changes
  const saveToLocal = useCallback(() => {
    if (typeof window === 'undefined') return;
    localStorage.setItem(PLAYGROUND_STORAGE_KEY, JSON.stringify({ nodes, edges, updatedAt: new Date().toISOString() }));
  }, [nodes, edges]);

  // Auto-save every time nodes/edges change
  useMemo(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(PLAYGROUND_STORAGE_KEY, JSON.stringify({ nodes, edges, updatedAt: new Date().toISOString() }));
    }
  }, [nodes, edges]);

  const onConnect = useCallback(
    (connection: Edge | Connection) => {
      setEdges((currentEdges) => addEdge({ ...connection, animated: false }, currentEdges));
    },
    [setEdges],
  );

  const onNodeClick = useCallback((_: MouseEvent, node: Node<BuilderNodeData>) => {
    setSelectedEdgeId(null);
    setSelectedNodeId(node.id);
  }, []);

  const onEdgeClick = useCallback((_: MouseEvent, edge: Edge) => {
    setSelectedNodeId(null);
    setSelectedEdgeId(edge.id);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedEdgeId(null);
    setSelectedNodeId(null);
  }, []);

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
    [setNodes],
  );

  const onDrop = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      const kind = event.dataTransfer.getData(DRAG_MIME_TYPE) as NodeKind;
      if (!kind) return;

      const instance = rfInstanceRef.current;
      const wrapper = wrapperRef.current;
      if (!instance || !wrapper) return;

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
      if (!selectedNodeId) return;
      setNodes((currentNodes) =>
        currentNodes.map((node) => (node.id === selectedNodeId ? { ...node, data: updater(node.data) } : node)),
      );
    },
    [selectedNodeId, setNodes],
  );

  const deleteSelected = useCallback(() => {
    if (selectedNodeId) {
      setNodes((current) => current.filter((n) => n.id !== selectedNodeId));
      setEdges((current) => current.filter((e) => e.source !== selectedNodeId && e.target !== selectedNodeId));
      setSelectedNodeId(null);
    } else if (selectedEdgeId) {
      setEdges((current) => current.filter((e) => e.id !== selectedEdgeId));
      setSelectedEdgeId(null);
    }
  }, [selectedNodeId, selectedEdgeId, setNodes, setEdges]);

  return (
    <div className="flex h-full">
      {/* Component Palette Sidebar */}
      {sidebarOpen && (
        <aside className="w-56 border-r bg-surface-elevated overflow-y-auto shrink-0">
          <div className="p-3 space-y-4">
            <p className="text-caption font-medium text-ink-secondary uppercase tracking-wider">Components</p>
            {NODE_KIND_GROUPS.map((group) => (
              <div key={group.title} className="space-y-1.5">
                <p className="text-caption text-ink-muted font-medium">{group.title}</p>
                {group.kinds.map((kind) => (
                  <div
                    key={kind}
                    draggable
                    onDragStart={(e) => {
                      e.dataTransfer.setData(DRAG_MIME_TYPE, kind);
                      e.dataTransfer.effectAllowed = 'copy';
                    }}
                    className="flex items-center gap-2 rounded-md border border-transparent px-2.5 py-1.5 text-caption cursor-grab hover:border-border hover:bg-surface active:cursor-grabbing transition-all"
                  >
                    <span className="w-2 h-2 rounded-full bg-accent/60 shrink-0" />
                    {NODE_LABELS[kind] ?? kind}
                  </div>
                ))}
              </div>
            ))}
          </div>
        </aside>
      )}

      {/* Canvas */}
      <div className="flex-1 relative" ref={wrapperRef}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          onEdgeClick={onEdgeClick}
          onPaneClick={onPaneClick}
          onDrop={onDrop}
          onDragOver={onDragOver}
          onInit={(instance) => { rfInstanceRef.current = instance; }}
          nodeTypes={nodeTypes}
          fitView
          deleteKeyCode={['Backspace', 'Delete']}
          className="bg-surface"
        >
          <Background gap={20} size={1} color="var(--color-border)" />
          <Controls className="!bg-surface-elevated !border-border" />
          <MiniMap
            className="!bg-surface-elevated !border-border"
            nodeColor="var(--color-accent)"
            maskColor="rgba(0,0,0,0.3)"
          />
        </ReactFlow>

        {/* Toggle sidebar button */}
        <button
          type="button"
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className="absolute top-3 left-3 z-10 rounded-md border bg-surface-elevated px-2 py-1 text-caption text-ink-secondary hover:text-ink transition-colors"
        >
          {sidebarOpen ? '← Hide' : '→ Palette'}
        </button>

        {/* Keyboard shortcut hint */}
        <div className="absolute bottom-3 right-3 z-10 text-caption text-ink-muted bg-surface-elevated/80 border rounded-md px-2 py-1">
          Drag components from palette • Delete/Backspace to remove
        </div>
      </div>

      {/* Node Inspector (right panel) */}
      {selectedNode && (
        <aside className="w-64 border-l bg-surface-elevated overflow-y-auto shrink-0 p-4 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-body-sm font-semibold">{NODE_LABELS[selectedNode.data.kind] ?? selectedNode.data.kind}</h3>
            <button
              type="button"
              onClick={deleteSelected}
              className="text-caption text-error hover:text-error/80 transition-colors"
            >
              Delete
            </button>
          </div>
          <div className="space-y-2">
            <label className="text-caption text-ink-secondary">Label</label>
            <input
              type="text"
              value={selectedNode.data.label}
              onChange={(e) => updateSelectedNode((d) => ({ ...d, label: e.target.value }))}
              className="w-full rounded-md border bg-surface px-2.5 py-1.5 text-body-sm focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <p className="text-caption text-ink-muted">
            ID: {selectedNode.id}
          </p>
        </aside>
      )}
    </div>
  );
}
