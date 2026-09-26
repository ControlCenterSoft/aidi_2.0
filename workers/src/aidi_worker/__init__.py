"""AIDI 2.0 worker package."""

from .contracts import (
    ContractError,
    DeploymentKind,
    FailureClass,
    FailureKind,
    FailureSignal,
    ModelCapability,
    ModelDeployment,
    RecoveryAction,
    RecoveryDecision,
    RecoveryEvidence,
    RetryContext,
    RoutingUnavailable,
    WorkerRequest,
    WorkerResult,
    classify_failure,
    decide_recovery,
    route_model,
)
from .worker import health

__all__ = [
    "ContractError",
    "DeploymentKind",
    "FailureClass",
    "FailureKind",
    "FailureSignal",
    "ModelCapability",
    "ModelDeployment",
    "RecoveryAction",
    "RecoveryDecision",
    "RecoveryEvidence",
    "RetryContext",
    "RoutingUnavailable",
    "WorkerRequest",
    "WorkerResult",
    "classify_failure",
    "decide_recovery",
    "health",
    "route_model",
]
