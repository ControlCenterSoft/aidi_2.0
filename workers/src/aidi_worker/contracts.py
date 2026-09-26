"""Release A worker, routing, and recovery foundation contracts.

These primitives are intentionally infrastructure-agnostic. They define policy and
validation rules only; concrete model runtimes, durable workflows, queues, and
persistence adapters are added in later slices.
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from typing import Iterable


class ContractError(ValueError):
    """Raised when a worker/recovery contract is internally inconsistent."""


class RoutingUnavailable(RuntimeError):
    """Raised when no qualified deployment satisfies a request."""


class ModelCapability(str, Enum):
    PLANNING = "planning"
    CODING = "coding"
    REVIEW = "review"
    DEBUGGING = "debugging"
    RECOVERY = "recovery"


class DeploymentKind(str, Enum):
    LOCAL = "local"
    CLOUD = "cloud"


class FailureClass(str, Enum):
    TASK = "task"
    PROJECT = "project"
    INFRASTRUCTURE = "infrastructure"
    CLOUD = "cloud"
    INTERNAL = "internal"


class FailureKind(str, Enum):
    DETERMINISTIC = "deterministic"
    TRANSIENT = "transient"
    OWNERSHIP_UNCERTAIN = "ownership_uncertain"
    POLICY_DENIED = "policy_denied"
    UNKNOWN = "unknown"


class RecoveryAction(str, Enum):
    RETRY = "retry"
    RECONCILE = "reconcile"
    NO_RETRY = "no_retry"
    ESCALATE = "escalate"


@dataclass(frozen=True, slots=True)
class ModelDeployment:
    deployment_id: str
    kind: DeploymentKind
    capabilities: frozenset[ModelCapability]
    qualified: bool
    healthy: bool = True

    def __post_init__(self) -> None:
        if not self.deployment_id.strip():
            raise ContractError("deployment_id is required")
        if not self.capabilities:
            raise ContractError("deployment must declare at least one capability")


@dataclass(frozen=True, slots=True)
class WorkerRequest:
    task_id: str
    attempt_id: str
    required_capabilities: frozenset[ModelCapability]
    require_local: bool = True
    strategy_revision: int = 0
    input_revision: int = 0

    def __post_init__(self) -> None:
        if not self.task_id.strip():
            raise ContractError("task_id is required")
        if not self.attempt_id.strip():
            raise ContractError("attempt_id is required")
        if not self.required_capabilities:
            raise ContractError("at least one required capability is required")
        if self.strategy_revision < 0 or self.input_revision < 0:
            raise ContractError("revisions cannot be negative")


@dataclass(frozen=True, slots=True)
class WorkerResult:
    task_id: str
    attempt_id: str
    success: bool
    evidence_refs: tuple[str, ...]
    output_ref: str | None = None

    def __post_init__(self) -> None:
        if not self.task_id.strip() or not self.attempt_id.strip():
            raise ContractError("task_id and attempt_id are required")
        if self.success and not self.evidence_refs:
            raise ContractError("successful worker result requires evidence")
        if any(not ref.strip() for ref in self.evidence_refs):
            raise ContractError("evidence references cannot be empty")


@dataclass(frozen=True, slots=True)
class FailureSignal:
    failure_class: FailureClass
    code: str
    message: str = ""
    ownership_uncertain: bool = False

    def __post_init__(self) -> None:
        if not self.code.strip():
            raise ContractError("failure code is required")


@dataclass(frozen=True, slots=True)
class RetryContext:
    attempt_number: int
    max_attempts: int = 3
    strategy_changed: bool = False
    input_changed: bool = False
    ownership_certain: bool = True

    def __post_init__(self) -> None:
        if self.attempt_number < 1:
            raise ContractError("attempt_number must be >= 1")
        if self.max_attempts < 1:
            raise ContractError("max_attempts must be >= 1")


@dataclass(frozen=True, slots=True)
class RecoveryDecision:
    action: RecoveryAction
    reason: str


@dataclass(frozen=True, slots=True)
class RecoveryEvidence:
    recovery_case_id: str
    attempt_id: str
    failure_class: FailureClass
    failure_kind: FailureKind
    action: RecoveryAction
    evidence_refs: tuple[str, ...]
    recovered: bool
    resumed: bool = False

    def __post_init__(self) -> None:
        if not self.recovery_case_id.strip() or not self.attempt_id.strip():
            raise ContractError("recovery_case_id and attempt_id are required")
        if any(not ref.strip() for ref in self.evidence_refs):
            raise ContractError("evidence references cannot be empty")
        if self.recovered and not self.evidence_refs:
            raise ContractError("recovered state requires evidence")
        if self.resumed and not self.recovered:
            raise ContractError("workflow cannot resume before recovery is proven")


_DETERMINISTIC_CODES = frozenset(
    {
        "aider_no_edit",
        "syntax_error",
        "parse_error",
        "compile_error",
        "build_error",
        "test_assertion",
        "invalid_contract",
    }
)
_TRANSIENT_CODES = frozenset(
    {
        "timeout",
        "rate_limit",
        "temporary_unavailable",
        "connection_reset",
        "runner_lost",
    }
)
_POLICY_CODES = frozenset({"policy_denied", "cloud_disabled", "budget_denied"})
_OWNERSHIP_CODES = frozenset({"lease_expired", "fencing_mismatch", "owner_unknown"})


def route_model(
    request: WorkerRequest,
    deployments: Iterable[ModelDeployment],
) -> ModelDeployment:
    """Return a qualified deployment while enforcing local-first semantics."""

    eligible = [
        deployment
        for deployment in deployments
        if deployment.qualified
        and deployment.healthy
        and request.required_capabilities.issubset(deployment.capabilities)
        and (not request.require_local or deployment.kind is DeploymentKind.LOCAL)
    ]
    if not eligible:
        raise RoutingUnavailable("no qualified deployment satisfies request")

    eligible.sort(
        key=lambda deployment: (
            deployment.kind is not DeploymentKind.LOCAL,
            deployment.deployment_id,
        )
    )
    return eligible[0]


def classify_failure(signal: FailureSignal) -> FailureKind:
    """Classify a normalized failure signal into recovery semantics."""

    code = signal.code.strip().lower()
    if signal.ownership_uncertain or code in _OWNERSHIP_CODES:
        return FailureKind.OWNERSHIP_UNCERTAIN
    if code in _POLICY_CODES:
        return FailureKind.POLICY_DENIED
    if code in _DETERMINISTIC_CODES:
        return FailureKind.DETERMINISTIC
    if code in _TRANSIENT_CODES:
        return FailureKind.TRANSIENT
    return FailureKind.UNKNOWN


def decide_recovery(kind: FailureKind, context: RetryContext) -> RecoveryDecision:
    """Choose a safe next action without blind retry loops."""

    if kind is FailureKind.OWNERSHIP_UNCERTAIN or not context.ownership_certain:
        return RecoveryDecision(
            RecoveryAction.RECONCILE,
            "ownership or lease state is uncertain; reconcile before side effects",
        )

    if kind is FailureKind.POLICY_DENIED:
        return RecoveryDecision(
            RecoveryAction.NO_RETRY,
            "policy denial requires policy/input change, not automatic retry",
        )

    if context.attempt_number >= context.max_attempts:
        return RecoveryDecision(
            RecoveryAction.ESCALATE,
            "retry budget exhausted",
        )

    if kind is FailureKind.DETERMINISTIC:
        if context.strategy_changed or context.input_changed:
            return RecoveryDecision(
                RecoveryAction.RETRY,
                "deterministic failure may retry only after strategy/input change",
            )
        return RecoveryDecision(
            RecoveryAction.NO_RETRY,
            "deterministic failure cannot blind-retry unchanged work",
        )

    if kind is FailureKind.TRANSIENT:
        return RecoveryDecision(
            RecoveryAction.RETRY,
            "transient failure is within retry budget",
        )

    return RecoveryDecision(
        RecoveryAction.ESCALATE,
        "unknown failure requires diagnosis before retry",
    )
