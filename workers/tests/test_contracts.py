from __future__ import annotations

import unittest

from aidi_worker.contracts import (
    ContractError,
    DeploymentKind,
    FailureClass,
    FailureKind,
    FailureSignal,
    ModelCapability,
    ModelDeployment,
    RecoveryAction,
    RecoveryEvidence,
    RetryContext,
    RoutingUnavailable,
    WorkerRequest,
    WorkerResult,
    classify_failure,
    decide_recovery,
    route_model,
)


class RoutingTests(unittest.TestCase):
    def setUp(self) -> None:
        self.local = ModelDeployment(
            deployment_id="local-coder",
            kind=DeploymentKind.LOCAL,
            capabilities=frozenset(
                {ModelCapability.CODING, ModelCapability.RECOVERY}
            ),
            qualified=True,
        )
        self.cloud = ModelDeployment(
            deployment_id="cloud-coder",
            kind=DeploymentKind.CLOUD,
            capabilities=frozenset(
                {ModelCapability.CODING, ModelCapability.RECOVERY}
            ),
            qualified=True,
        )

    def test_local_is_preferred_even_when_cloud_is_allowed(self) -> None:
        request = WorkerRequest(
            task_id="T-1",
            attempt_id="A-1",
            required_capabilities=frozenset({ModelCapability.CODING}),
            require_local=False,
        )
        self.assertEqual(route_model(request, [self.cloud, self.local]), self.local)

    def test_local_required_never_falls_back_to_cloud(self) -> None:
        request = WorkerRequest(
            task_id="T-1",
            attempt_id="A-1",
            required_capabilities=frozenset({ModelCapability.CODING}),
            require_local=True,
        )
        with self.assertRaises(RoutingUnavailable):
            route_model(request, [self.cloud])

    def test_unqualified_model_is_never_selected(self) -> None:
        request = WorkerRequest(
            task_id="T-1",
            attempt_id="A-1",
            required_capabilities=frozenset({ModelCapability.CODING}),
        )
        unqualified = ModelDeployment(
            deployment_id="local-unqualified",
            kind=DeploymentKind.LOCAL,
            capabilities=frozenset({ModelCapability.CODING}),
            qualified=False,
        )
        with self.assertRaises(RoutingUnavailable):
            route_model(request, [unqualified])


class RecoveryTests(unittest.TestCase):
    def test_lease_uncertainty_reconciles_instead_of_retry(self) -> None:
        signal = FailureSignal(
            failure_class=FailureClass.INFRASTRUCTURE,
            code="lease_expired",
        )
        kind = classify_failure(signal)
        decision = decide_recovery(kind, RetryContext(attempt_number=1))
        self.assertEqual(kind, FailureKind.OWNERSHIP_UNCERTAIN)
        self.assertEqual(decision.action, RecoveryAction.RECONCILE)

    def test_deterministic_failure_does_not_blind_retry(self) -> None:
        kind = classify_failure(
            FailureSignal(failure_class=FailureClass.TASK, code="syntax_error")
        )
        decision = decide_recovery(kind, RetryContext(attempt_number=1))
        self.assertEqual(kind, FailureKind.DETERMINISTIC)
        self.assertEqual(decision.action, RecoveryAction.NO_RETRY)

    def test_deterministic_failure_can_retry_after_changed_strategy(self) -> None:
        decision = decide_recovery(
            FailureKind.DETERMINISTIC,
            RetryContext(attempt_number=1, strategy_changed=True),
        )
        self.assertEqual(decision.action, RecoveryAction.RETRY)

    def test_transient_failure_stops_after_retry_budget(self) -> None:
        decision = decide_recovery(
            FailureKind.TRANSIENT,
            RetryContext(attempt_number=3, max_attempts=3),
        )
        self.assertEqual(decision.action, RecoveryAction.ESCALATE)

    def test_unknown_failure_escalates(self) -> None:
        kind = classify_failure(
            FailureSignal(failure_class=FailureClass.INTERNAL, code="novel_failure")
        )
        self.assertEqual(kind, FailureKind.UNKNOWN)
        self.assertEqual(
            decide_recovery(kind, RetryContext(attempt_number=1)).action,
            RecoveryAction.ESCALATE,
        )


class EvidenceTests(unittest.TestCase):
    def test_successful_worker_result_requires_evidence(self) -> None:
        with self.assertRaises(ContractError):
            WorkerResult("T-1", "A-1", True, ())

    def test_recovered_case_requires_evidence(self) -> None:
        with self.assertRaises(ContractError):
            RecoveryEvidence(
                recovery_case_id="R-1",
                attempt_id="A-1",
                failure_class=FailureClass.TASK,
                failure_kind=FailureKind.DETERMINISTIC,
                action=RecoveryAction.RETRY,
                evidence_refs=(),
                recovered=True,
            )

    def test_resume_requires_proven_recovery(self) -> None:
        with self.assertRaises(ContractError):
            RecoveryEvidence(
                recovery_case_id="R-1",
                attempt_id="A-1",
                failure_class=FailureClass.TASK,
                failure_kind=FailureKind.TRANSIENT,
                action=RecoveryAction.RETRY,
                evidence_refs=("ci://run/123",),
                recovered=False,
                resumed=True,
            )


if __name__ == "__main__":
    unittest.main()
