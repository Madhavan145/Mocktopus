import type { LiveMetricsSnapshot, SimulationRunRecord } from './types';

type LiveSnapshotMessage = {
  projectId: string;
  snapshot: LiveMetricsSnapshot | null;
};

type ActiveRunMessage = {
  projectId: string;
  runId: string | null;
};

export function simulationLastRunKey(projectId: string): string {
  return `moktopus:project:${projectId}:simulation:last`;
}

export function simulationHistoryKey(projectId: string): string {
  return `moktopus:project:${projectId}:simulation:history`;
}

export function simulationLiveSnapshotKey(projectId: string): string {
  return `moktopus:project:${projectId}:simulation:live`;
}

export function simulationActiveRunKey(projectId: string): string {
  return `moktopus:project:${projectId}:simulation:active-run`;
}

export function writeSimulationRun(projectId: string, run: SimulationRunRecord): void {
  localStorage.setItem(simulationLastRunKey(projectId), JSON.stringify(run));

  const rawHistory = localStorage.getItem(simulationHistoryKey(projectId));
  let history: SimulationRunRecord[] = [];
  if (rawHistory) {
    try {
      history = JSON.parse(rawHistory) as SimulationRunRecord[];
    } catch {
      history = [];
    }
  }

  history.unshift(run);
  if (history.length > 10) {
    history = history.slice(0, 10);
  }

  localStorage.setItem(simulationHistoryKey(projectId), JSON.stringify(history));
}

export function readLastSimulationRun(projectId: string): SimulationRunRecord | null {
  const raw = localStorage.getItem(simulationLastRunKey(projectId));
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as SimulationRunRecord;
  } catch {
    return null;
  }
}

export function readSimulationHistory(projectId: string): SimulationRunRecord[] {
  const raw = localStorage.getItem(simulationHistoryKey(projectId));
  if (!raw) {
    return [];
  }

  try {
    return JSON.parse(raw) as SimulationRunRecord[];
  } catch {
    return [];
  }
}

export function readLiveSimulationSnapshot(projectId: string): LiveMetricsSnapshot | null {
  const raw = localStorage.getItem(simulationLiveSnapshotKey(projectId));
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as LiveMetricsSnapshot;
  } catch {
    return null;
  }
}

export function publishLiveSimulationSnapshot(projectId: string, snapshot: LiveMetricsSnapshot | null): void {
  const key = simulationLiveSnapshotKey(projectId);

  if (snapshot) {
    localStorage.setItem(key, JSON.stringify(snapshot));
  } else {
    localStorage.removeItem(key);
  }

  if (typeof window === 'undefined' || typeof BroadcastChannel === 'undefined') {
    return;
  }

  const channel = new BroadcastChannel('moktopus-simulation-live');
  const message: LiveSnapshotMessage = { projectId, snapshot };
  channel.postMessage(message);
  channel.close();
}

export function subscribeLiveSimulationSnapshot(
  projectId: string,
  onSnapshot: (snapshot: LiveMetricsSnapshot | null) => void,
): () => void {
  if (typeof window === 'undefined' || typeof BroadcastChannel === 'undefined') {
    return () => undefined;
  }

  const channel = new BroadcastChannel('moktopus-simulation-live');
  channel.onmessage = (event: MessageEvent<LiveSnapshotMessage>) => {
    if (event.data?.projectId !== projectId) {
      return;
    }
    onSnapshot(event.data.snapshot ?? null);
  };

  return () => channel.close();
}

export function readActiveSimulationRunId(projectId: string): string | null {
  return localStorage.getItem(simulationActiveRunKey(projectId));
}

export function publishActiveSimulationRunId(projectId: string, runId: string | null): void {
  const key = simulationActiveRunKey(projectId);
  if (runId) {
    localStorage.setItem(key, runId);
  } else {
    localStorage.removeItem(key);
  }

  if (typeof window === 'undefined' || typeof BroadcastChannel === 'undefined') {
    return;
  }

  const channel = new BroadcastChannel('moktopus-simulation-run-id');
  const message: ActiveRunMessage = { projectId, runId };
  channel.postMessage(message);
  channel.close();
}

export function subscribeActiveSimulationRunId(
  projectId: string,
  onRunId: (runId: string | null) => void,
): () => void {
  if (typeof window === 'undefined' || typeof BroadcastChannel === 'undefined') {
    return () => undefined;
  }

  const channel = new BroadcastChannel('moktopus-simulation-run-id');
  channel.onmessage = (event: MessageEvent<ActiveRunMessage>) => {
    if (event.data?.projectId !== projectId) {
      return;
    }
    onRunId(event.data.runId ?? null);
  };

  return () => channel.close();
}
