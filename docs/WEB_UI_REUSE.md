# Web UI Reuse Strategy

Status: implementation started

## Goal

Evolve the current operational Web UI and the AIDI 2.0 product UI in parallel without coupling canonical execution state to a specific frontend.

## Reuse boundary

Reusable across current UI, AIDI 2.0 and AIDI 2.1:
- design tokens and status semantics;
- primary/secondary navigation patterns;
- metric cards, panels, badges, tables and charts;
- project/incident/recovery presentation components;
- responsive layout rules.

Not reusable as UI state:
- Task/Attempt ownership;
- Project/Release lifecycle;
- Recovery decisions;
- authoritative progress;
- security or access decisions.

Those values must come from canonical backend contracts.

## Promotion path

Prototype -> type/build checks -> Shadow -> Canary -> Promote -> Observe.

The current live UI remains untouched until a component or route has a compatible API contract and passes its release gates.

## Version scope

AIDI 2.0 owns the product workspace, project lifecycle, Super Admin and AIDI Orb surfaces defined by SPEC.

AIDI 2.1 may build Doctor RCA/Learning/Training Lab surfaces on this same component foundation as a reversible overlay. The installed product remains 2.0 until the 2.0 gates and later the complete 2.1 acceptance gates pass.
