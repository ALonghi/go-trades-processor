"use client";

export function cn(...c: Array<string | false | null | undefined>) {
  return c.filter(Boolean).join(" ");
}

export function GlassCard(props: React.HTMLAttributes<HTMLDivElement>) {
  const { className, ...rest } = props;
  return <div className={cn("glass p-4 sm:p-6", className)} {...rest} />;
}

export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { label: string; value: T }[];
  value: T;
  onChange: (v: T) => void;
}) {
  return (
    <div className="glass inline-flex overflow-hidden rounded-xl">
      {options.map((o) => {
        const active = o.value === value;
        return (
          <button
            key={o.value}
            onClick={() => onChange(o.value)}
            className={cn(
              "px-3 py-1.5 text-sm transition-colors",
              active ? "bg-white/30 text-black" : "text-white/80 hover:bg-white/10"
            )}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

export function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="glass p-4">
      <div className="text-xs uppercase tracking-wide text-white/60">{label}</div>
      <div className="mt-1 text-2xl font-semibold">{value}</div>
    </div>
  );
}
