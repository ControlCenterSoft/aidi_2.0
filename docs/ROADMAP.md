> Canonical source: AIDI v2.0.0 — Full Implementation Roadmap (APPROVED)
> Source: https://docs.google.com/document/d/1ZpVvWWx9qqDze6szlxeIuCS4Eqv4jKOKpAx96GL1duU/edit?usp=drivesdk
> Imported: 2026-09-26
> Status: APPROVED

﻿AIDI v2.0.0 — Full Implementation Roadmap
Status: APPROVED · Decomposition A–K + RC: 100%
1. Общий порядок релизов
A Foundation → B Identity & Platform → C Infrastructure → D AI Platform → E Specification & Planning → F Autonomous Development → G Recovery & Self-Healing → H Self-Development & Update → I CRM / Client Integration → J Storage / Supply Chain / Compliance → K HA / DR / Hardening → RC Final Acceptance.
2. Объём backlog
* Release A — 86 implementation Issues, 15 Gates.
* Release B — 127 implementation Issues, 16 Gates. Ранее фигурировавшее число 119 было арифметической ошибкой; scope не изменён.
* Release C — 133 implementation Issues, 20 Gates.
* Release D — 145 implementation Issues, 22 Gates.
* Release E — 140 implementation Issues, 20 Gates.
* Release F — 152 implementation Issues, 20 Gates.
* Release G — 137 implementation Issues, 20 Gates.
* Release H — 100 implementation Issues, 17 Gates.
* Release I — 104 implementation Issues, 17 Gates.
* Release J — 133 implementation Issues, 20 Gates.
* Release K — 133 implementation Issues, 22 Gates.
* Release RC — 80 implementation Issues, 22 Gates.
Итого: 1470 implementation Issues + 231 Gate Issues = 1701 tracker cards. Master graph содержит 939 hard dependency edges; duplicate stable keys = 0; unresolved dependencies = 0; dependency cycles = 0.
3. Promotion policy
* Следующий Release не становится authoritative execution scope до PASS prerequisite gate предыдущего Release.
* Внутри Release допускается параллельная работа только по dependency-ready Issues.
* Issue получает DONE только при code/config + applicable tests + review + docs/contracts + evidence.
* Gate Issue не содержит production implementation и закрывается только после PASS связанного test/evidence.
4. Cutover activation
* После импорта новой roadmap в текущую AIDI активируется только dependency-ready queue Release A.
* B–RC остаются BLOCKED prerequisite gates.
* Legacy PRE-2.0 backlog сохраняется исторически, но исключается из новой execution queue.