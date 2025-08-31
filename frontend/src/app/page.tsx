"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { GlassCard, Segmented, Stat, cn } from "./(components)/ui";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type Holding = {
  entity: "zurich" | "new_york";
  instrument_type: "stock" | "crypto";
  symbol: string;
  quantity: number;
};
type Entity = "zurich" | "new_york" | "all";

type EntityOption = {
  value: Entity,
  label: string

}

const ENTITY_OPTIONS: EntityOption[] = [
  { value: "zurich", label: "Zurich" },
  { value: "new_york", label: "New York" },
  { value: "all", label: "All" },
] as const;

const fmtQty = new Intl.NumberFormat(undefined, { maximumFractionDigits: 8 });

function usePolling<T>(fn: () => Promise<T>, deps: unknown[] = [], ms = 5000) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const t = useRef<number | null>(null);

  async function run() {
    setLoading(true);
    setError(null);
    try {
      const res = await fn();
      setData(res);
    } catch (e) {
      setError(e?.message ?? "Failed to load");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    const tick = async () => {
      if (!active) return;
      await run();
      const delay = ms + Math.floor(Math.random() * 500);
      t.current = window.setTimeout(tick, delay) as unknown as number;
    };
    const onVis = () => (document.hidden ? t.current && clearTimeout(t.current) : tick());
    document.addEventListener("visibilitychange", onVis);
    tick();
    return () => {
      active = false;
      document.removeEventListener("visibilitychange", onVis);
      if (t.current) clearTimeout(t.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error, reload: run };
}

export default function Page() {
  const [entity, setEntity] = useState<Entity>("all");

  const { data, loading, error, reload } = usePolling<Holding[]>(
    async () => {
      const url = entity === "all" ? `${API}/api/holdings` : `${API}/api/holdings/${entity}`;
      const r = await fetch(url, { cache: "no-store" });
      if (!r.ok) throw new Error(await r.text());
      const json = await r.json();
      return Array.isArray(json) ? (json as Holding[]) : [];
    },
    [entity],
    5000
  );

  const rows = data ?? [];
  const grouped = useMemo(() => {
    const g: Record<"stock" | "crypto", Holding[]> = { stock: [], crypto: [] };
    for (const h of rows) g[h.instrument_type].push(h);
    return g;
  }, [rows]);

  return (
    <div className="space-y-6">
      {/* header */}
      <GlassCard className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">Dashboard</h1>
          <button
            onClick={reload}
            className="cursor-pointer rounded-xl border border-white/20 px-3 py-1 text-sm text-white/90 hover:bg-white/10"
          >
            Refresh
          </button>
        </div>
        <Segmented
          options={ENTITY_OPTIONS}
          value={entity}
          onChange={(v) => setEntity(v as Entity)}
        />
      </GlassCard>

      {/* quick stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Stat label="Positions" value={String(rows.length)} />
        <Stat label="Stocks" value={String(grouped.stock.length)} />
        <Stat label="Crypto" value={String(grouped.crypto.length)} />
      </div>

      {/* data blocks */}
      {(["stock", "crypto"] as const).map((itype) => {
        const items = grouped[itype];
        return (
          <GlassCard key={itype}>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-lg font-medium capitalize">{itype} holdings</h2>
              <span className="text-xs rounded-full bg-white/10 px-2 py-0.5">
                {items.length} row{items.length === 1 ? "" : "s"}
              </span>
            </div>
            <div className="overflow-x-auto">
              <table className="table-glass w-full text-sm">
                <thead>
                  <tr>
                    <th className="py-2 text-left">Entity</th>
                    <th className="text-left">Symbol</th>
                    <th className="text-right pr-1">Quantity</th>
                  </tr>
                </thead>
                <tbody>
                  {loading && items.length === 0 ? (
                    <SkeletonRows />
                  ) : items.length ? (
                    items.map((h) => (
                      <tr key={`${h.entity}-${h.instrument_type}-${h.symbol}`}>
                        <td className="py-2 capitalize">{h.entity.replace("_", " ")}</td>
                        <td className="font-medium">{h.symbol}</td>
                        <td className="text-right tabular-nums">{fmtQty.format(h.quantity)}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={3} className="py-6 text-white/70">
                        No {itype} holdings yet.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            <div className="divider mt-4" />
            <div className="mt-3 text-xs text-white/70">
              {error ? <span className="text-red-400">Error: {error}</span> : loading ? "Refreshing…" : "Live"}
            </div>
          </GlassCard>
        );
      })}
    </div>
  );
}

function SkeletonRows({ rows = 5 }: { rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i}>
          <td className="py-2">
            <span className="inline-block h-4 w-24 animate-pulse rounded bg-white/10" />
          </td>
          <td>
            <span className="inline-block h-4 w-16 animate-pulse rounded bg-white/10" />
          </td>
          <td className="text-right">
            <span className="inline-block h-4 w-20 animate-pulse rounded bg-white/10" />
          </td>
        </tr>
      ))}
    </>
  );
}
