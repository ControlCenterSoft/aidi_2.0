# Release A Public API mutation command contract

Status: IN DEVELOPMENT

This slice defines a transport-independent mutation command for existing canonical state.

Required metadata:
- command_id: stable identity for later idempotency handling
- correlation_id: trace/error correlation
- target kind and id
- operation
- expected_revision: optimistic-concurrency input

The contract is separate from orchestration.WorkflowCommand.

This slice supports SPEC section 13.1 and provides prerequisites for AC-STATE-004 and AC-STATE-005. It does not implement deduplication, authoritative revision comparison, HTTP routing, persistence, or execution.

Tests cover the valid command and every required invariant.
