"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { GlassCard, Segmented, cn } from "../(components)/ui";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type Row = [
  tradeId: string,
  entity: string,
  itype: string,
  symbol: string,
  qty: number,
  price: number | null,
  ts: string,
];
type Entity = "" | "zurich" | "new_york";
const ENTITY_OPTIONS: Array<{ value: Entity; label: string }> = [
  { value: "", label: "All" },
  { value: "zurich", label: "Zurich" },
  { value: "new_york", label: "New York" },
];

const fmtQty = new Intl.NumberFormat(undefined, { maximumFractionDigits: 8 });
const fmtPrice = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 8,
});

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
    const onVis = () =>
      document.hidden ? t.current && clearTimeout(t.current) : tick();
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

function fromNow(iso: string) {
  const t = new Date(iso).getTime();
  const mins = Math.round((Date.now() - t) / 60000);
  if (Math.abs(mins) < 1) return "just now";
  if (Math.abs(mins) < 60) return `${mins}m ago`;
  const hrs = Math.round(mins / 60);
  if (Math.abs(hrs) < 24) return `${hrs}h ago`;
  const days = Math.round(hrs / 24);
  return `${days}d ago`;
}

export default function Trades() {
  const [entity, setEntity] = useState<Entity>("");
  const [symbolFilter, setSymbolFilter] = useState("");

  const { data, loading, error, reload } = usePolling<Row[]>(
    async () => {
      const url = new URL(`${API}/api/trades`);
      url.searchParams.set("limit", "200");
      if (entity) url.searchParams.set("entity", entity);
      const r = await fetch(url, { cache: "no-store" });
      if (!r.ok) throw new Error(await r.text());
      const j = await r.json();
      const rows: unknown = j?.rows;
      return Array.isArray(rows) ? (rows as Row[]) : [];
    },
    [entity],
    5000,
  );

  const rows = useMemo(() => {
    const base = data ?? [];
    const f = symbolFilter.trim().toUpperCase();
    return f ? base.filter((r) => r[3].toUpperCase().includes(f)) : base;
  }, [data, symbolFilter]);

  return (
    <div className="space-y-6">
      <GlassCard className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">Trades</h1>
          <button
            onClick={reload}
            className="cursor-pointer rounded-xl border border-white/20 px-3 py-1 text-sm text-white/90 hover:bg-white/10"
          >
            Refresh
          </button>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Segmented
            options={ENTITY_OPTIONS}
            value={entity}
            onChange={(v) => setEntity(v as Entity)}
          />
          <input
            value={symbolFilter}
            onChange={(e) => setSymbolFilter(e.target.value)}
            placeholder="Filter by symbol…"
            className="glass px-3 py-1.5 text-sm outline-none w-48"
          />
        </div>

        <div className="text-xs text-white/70">
          {loading ? (
            "Refreshing…"
          ) : error ? (
            <span className="text-red-300">Error: {error}</span>
          ) : (
            `${rows.length} row${rows.length === 1 ? "" : "s"}`
          )}
        </div>
      </GlassCard>

      <GlassCard className="p-0 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="table-glass w-full text-sm">
            <thead>
              <tr>
                <th className="py-2 pl-4 text-left">Trade ID</th>
                <th className="text-left">Entity</th>
                <th className="text-left">Type</th>
                <th className="text-left">Symbol</th>
                <th className="text-right">Qty</th>
                <th className="text-right">Price</th>
                <th className="pr-4 text-left">Timestamp</th>
              </tr>
            </thead>
            <tbody>
              {loading && (data ?? []).length === 0 ? (
                <SkeletonRows />
              ) : rows.length ? (
                rows.map((r) => (
                  <tr key={r[0]} className="hover:bg-white/5">
                    <td className="py-2 pl-4">
                      <code className="rounded bg-white/10 px-1.5 py-0.5 text-[11px]">
                        {r[0]}
                      </code>
                      <button
                        onClick={() =>
                          navigator.clipboard.writeText(r[0]).catch(() => {})
                        }
                        className="ml-2 text-xs text-white/70 underline-offset-2 hover:underline"
                        title="Copy trade ID"
                      >
                        Copy
                      </button>
                    </td>
                    <td className="capitalize">{r[1].replace("_", " ")}</td>
                    <td>
                      <span
                        className={cn(
                          "inline-block rounded-full border px-2 py-0.5 text-xs",
                          r[2] === "crypto"
                            ? "border-fuchsia-300/50"
                            : "border-sky-300/50",
                        )}
                      >
                        {r[2]}
                      </span>
                    </td>
                    <td className="font-medium">{r[3]}</td>
                    <td className="text-right tabular-nums">
                      {fmtQty.format(r[4])}
                    </td>
                    <td className="text-right tabular-nums">
                      {r[5] == null ? "-" : fmtPrice.format(r[5])}
                    </td>
                    <td className="pr-4">
                      <div className="text-xs text-white/80">
                        {new Date(r[6]).toLocaleString()}
                      </div>
                      <div className="text-[10px] text-white/60">
                        {fromNow(r[6])}
                      </div>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={7} className="py-8 text-center text-white/70">
                    No trades yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </GlassCard>
    </div>
  );
}

function SkeletonRows({ rows = 8 }: { rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i} className="border-t border-white/10">
          {Array.from({ length: 7 }).map((__, j) => (
            <td key={j} className={cn("py-2", j === 0 ? "pl-4" : "")}>
              <span className="inline-block h-4 w-24 animate-pulse rounded bg-white/10" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}
