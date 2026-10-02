# Release A Public API mutation command contract

Status: IN DEVELOPMENT

This slice defines a transport-independent mutation command for existing canonical state.

Required metadata:
- command_id: stable identity for idempotency handling
- correlation_id: trace/error correlation
- target kind and id
- operation
- expected_revision: optimistic-concurrency input

The contract is separate from orchestration.WorkflowCommand.

## Foundation Command endpoint (A10-004)

`internal/commandendpoint` composes the existing Release A command, authorization and idempotency contracts into one fail-closed execution path:

1. validate `apicommand.MutationCommand`;
2. resolve effective server-side authorization inputs through `AuthorizationRequestBuilder`;
3. enforce `authzpipeline.Authorize` (RBAC -> policy -> active tool);
4. only after authorization succeeds, enter `inbox.Store.Once` using `CommandID`;
5. invoke the injected executor only for the first logical execution;
6. return the previously recorded result for duplicate Command IDs without repeating the mutation.

Authorization is deliberately evaluated **before** the Inbox lookup. A caller therefore cannot replay a previously successful Command ID after losing authorization and receive the stored result through this endpoint.

The HTTP handler accepts only the command envelope. Effective Policy/Role/Tool inputs are not accepted from the request body; runtime wiring owns that lookup through `AuthorizationRequestBuilder`. The handler returns `403` on an authorization/policy denial and does not record the denied Command ID in the Inbox.

The endpoint remains infrastructure-neutral in Release A: concrete authentication/session wiring, persisted policy lookup, durable Inbox storage and operation-specific mutation executors are injected by later runtime/adaptor slices.

## Evidence

`internal/commandendpoint/commandendpoint_test.go` covers:
- the same Command ID invoking the executor exactly once and returning the same stored logical result on replay;
- RBAC denial before Inbox/executor;
- explicit policy DENY before Inbox/executor;
- re-authorization of a duplicate Command ID;
- HTTP POST success/duplicate behavior;
- HTTP rejection of denied and non-POST requests.

This slice supports SPEC section 13.1 and closes the A10-004 acceptance criteria by wiring already-delivered A3-004 Inbox/idempotency and A5-005 Authorization -> Policy -> Execution contracts into the Foundation Command path. Authoritative revision comparison and operation-specific persistence remain separate concerns.

Tests cover the valid command metadata contract and every required endpoint invariant.
