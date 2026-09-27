import type { PropsWithChildren, ReactNode } from "react";

export type AidiStatus = "ok" | "active" | "warning" | "error";

export function StatusBadge({
  status,
  children,
}: PropsWithChildren<{ status: AidiStatus }>) {
  return <span className={`aidi-badge aidi-badge--${status}`}>● {children}</span>;
}

export function MetricCard({
  label,
  value,
  note,
}: {
  label: string;
  value: ReactNode;
  note?: ReactNode;
}) {
  return (
    <section className="aidi-card">
      <p className="aidi-card-note">{label}</p>
      <div className="aidi-metric-value">{value}</div>
      {note ? <div className="aidi-metric-foot">{note}</div> : null}
    </section>
  );
}

export function Panel({
  title,
  note,
  children,
  wide = false,
  full = false,
}: PropsWithChildren<{
  title: string;
  note?: string;
  wide?: boolean;
  full?: boolean;
}>) {
  const modifier = full ? " aidi-card--full" : wide ? " aidi-card--wide" : "";
  return (
    <section className={`aidi-card${modifier}`}>
      <div className="aidi-section-title">
        <div>
          <h2>{title}</h2>
          {note ? <p className="aidi-card-note">{note}</p> : null}
        </div>
      </div>
      {children}
    </section>
  );
}

export function Navigation({
  items,
  current,
  level = "primary",
  onChange,
}: {
  items: readonly string[];
  current: string;
  level?: "primary" | "secondary";
  onChange: (item: string) => void;
}) {
  return (
    <nav className={level === "primary" ? "aidi-primary-nav" : "aidi-secondary-nav"}>
      {items.map((item) => (
        <button
          type="button"
          className="aidi-nav-button"
          key={item}
          aria-current={item === current ? "page" : undefined}
          onClick={() => onChange(item)}
        >
          {item}
        </button>
      ))}
    </nav>
  );
}
