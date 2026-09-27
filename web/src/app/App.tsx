import { useMemo, useState } from "react";
import { MetricCard, Navigation, Panel, StatusBadge } from "../ui/components";

const PRIMARY = ["Projects", "Action Center", "Notifications", "Files", "Profile"] as const;
const PROJECT = ["Overview", "ТЗ", "Plan", "Kanban", "Releases", "Results", "Files", "History", "Participants"] as const;

export default function App() {
  const [primary, setPrimary] = useState<string>("Projects");
  const [projectSection, setProjectSection] = useState<string>("Overview");

  const path = useMemo(() => {
    if (primary !== "Projects") return primary;
    return `Projects / AIDI 2.0 / ${projectSection}`;
  }, [primary, projectSection]);

  return (
    <div className="aidi-shell">
      <header className="aidi-header">
        <div>
          <h1 className="aidi-title">AIDI 2.0</h1>
          <div className="aidi-path">{path}</div>
        </div>
        <div className="aidi-meta">Reusable Product UI Foundation · feature branch</div>
      </header>

      <Navigation items={PRIMARY} current={primary} onChange={setPrimary} />
      {primary === "Projects" ? (
        <Navigation items={PROJECT} current={projectSection} level="secondary" onChange={setProjectSection} />
      ) : null}

      <main className="aidi-main">
        {primary === "Projects" && projectSection === "Overview" ? <ProjectOverview /> : (
          <Panel title={projectSection === "Overview" ? primary : projectSection} note="Foundation route prepared for canonical AIDI 2.0 data contracts." full>
            <p className="aidi-card-note">
              This screen intentionally contains no fake progress or local-only state. It will be connected to canonical entities as the corresponding release scope becomes available.
            </p>
          </Panel>
        )}
      </main>
    </div>
  );
}

function ProjectOverview() {
  return (
    <div className="aidi-grid">
      <MetricCard label="Готовность проекта" value="—" note="canonical project progress" />
      <MetricCard label="Активные задачи" value="—" note="Task / Attempt ownership" />
      <MetricCard label="Ошибки" value="—" note="Action Center projection" />
      <MetricCard label="Release" value="A" note="Foundation" />
      <MetricCard label="ETA" value="—" note="calculated from canonical execution data" />

      <Panel title="Project lifecycle" note="AIDI 2.0 canonical product flow" wide>
        <div className="aidi-badges">
          <StatusBadge status="active">Specification</StatusBadge>
          <StatusBadge status="active">Planning</StatusBadge>
          <StatusBadge status="warning">Autonomous development</StatusBadge>
          <StatusBadge status="warning">Release gates</StatusBadge>
        </div>
      </Panel>

      <Panel title="Reusable UI contract" note="Shared with current Web UI and future AIDI 2.1" wide>
        <div className="aidi-list">
          <div className="aidi-list-row"><div className="aidi-list-label">Status</div><div>StatusBadge: ok / active / warning / error</div></div>
          <div className="aidi-list-row"><div className="aidi-list-label">Metrics</div><div>MetricCard for progress, resources, errors and cost</div></div>
          <div className="aidi-list-row"><div className="aidi-list-label">Navigation</div><div>Primary + project-level navigation</div></div>
        </div>
      </Panel>

      <Panel title="Compatibility" note="No production cutover in this change" full>
        <div className="aidi-badges">
          <StatusBadge status="ok">Current live UI untouched</StatusBadge>
          <StatusBadge status="ok">Design System V3 aligned</StatusBadge>
          <StatusBadge status="active">React foundation</StatusBadge>
          <StatusBadge status="active">AIDI 2.1 reusable</StatusBadge>
        </div>
      </Panel>
    </div>
  );
}
