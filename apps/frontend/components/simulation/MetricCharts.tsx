import type { MetricsDerivedPoint, SimulationEvent } from './types';

type TrendChartProps = {
  title: string;
  colorClassName: string;
  series: MetricsDerivedPoint[];
  accessor: (point: MetricsDerivedPoint) => number;
  formatter?: (value: number) => string;
};

export function TrendChart({ title, colorClassName, series, accessor, formatter }: TrendChartProps) {
  if (!series.length) {
    return (
      <article className="rounded-lg border bg-black/30 p-3">
        <h4 className="text-sm font-medium">{title}</h4>
        <p className="mt-2 text-xs text-mute">No series data for this run yet.</p>
      </article>
    );
  }

  const values = series.map(accessor);
  const maxValue = Math.max(...values, 1);
  const minValue = Math.min(...values, 0);
  const span = Math.max(maxValue - minValue, 1);

  const points = values
    .map((value, index) => {
      const x = (index / Math.max(values.length - 1, 1)) * 100;
      const normalized = (value - minValue) / span;
      const y = 100 - normalized * 100;
      return `${x},${y}`;
    })
    .join(' ');

  const latest = values[values.length - 1] ?? 0;

  return (
    <article className="rounded-lg border bg-black/30 p-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium">{title}</h4>
        <span className="text-xs text-mute">{formatter ? formatter(latest) : latest.toFixed(2)}</span>
      </div>
      <svg className="mt-2 h-24 w-full" viewBox="0 0 100 100" preserveAspectRatio="none" role="img" aria-label={title}>
        <polyline fill="none" stroke="currentColor" strokeWidth="2" points={points} className={colorClassName} />
      </svg>
    </article>
  );
}

type EventMixChartProps = {
  events: SimulationEvent[];
};

export function EventMixChart({ events }: EventMixChartProps) {
  const counts = events.reduce<Record<string, number>>((acc, event) => {
    acc[event.type] = (acc[event.type] ?? 0) + 1;
    return acc;
  }, {});

  const entries = Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6);

  const total = entries.reduce((sum, [, count]) => sum + count, 0);

  return (
    <article className="rounded-lg border bg-black/30 p-3">
      <h4 className="text-sm font-medium">Event Mix</h4>
      {!entries.length ? (
        <p className="mt-2 text-xs text-mute">No events captured.</p>
      ) : (
        <div className="mt-3 space-y-2">
          {entries.map(([name, count]) => {
            const percent = total > 0 ? (count / total) * 100 : 0;
            return (
              <div key={name}>
                <div className="flex items-center justify-between text-xs text-mute">
                  <span>{name}</span>
                  <span>{count}</span>
                </div>
                <div className="mt-1 h-2 rounded bg-black/40">
                  <div className="h-2 rounded bg-accent" style={{ width: `${percent}%` }} />
                </div>
              </div>
            );
          })}
        </div>
      )}
    </article>
  );
}
