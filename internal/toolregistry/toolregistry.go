// Package toolregistry defines the transport-independent Tool Registry
// domain contract required by SPEC §8.4 (Software and Service Catalog) and
// §17.3 (Tool Broker), as the dependency-ready next Release A Foundation
// slice (backlog A5-004, depends only on completed A1-001).
//
// It defines pure domain types and invariants: a ToolID identifier (aligned
// with internal/canonical identifier conventions), a Tool entity (name,
// version, license, closed-set Status, resource needs, and a dependency
// list), a closed ProvisionPolicy enum, and an in-memory, pure-function
// Registry aggregate enforcing ToolID uniqueness and field validity.
// Validation errors are always wrapped sentinel errors, never panics.
//
// This package deliberately excludes Tool Broker enforcement (actual
// admission/authorization of tool usage), any UI/catalog presentation,
// catalog persistence, and runtime process/VM/queue/runner execution of a
// tool. Those are later Release A/B slices layered on top of this contract.
// No file in this package touches HTTP handlers, PostgreSQL, NATS,
// Temporal, Forgejo, or any VM/runner/queue integration, and the package
// introduces no new external module dependencies.
package toolregistry

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ToolID is the canonical identifier of a Tool, aligned with the
// internal/canonical identifier conventions (a non-empty, non-blank
// string).
type ToolID string

// Validate rejects an empty or whitespace-only ToolID.
func (id ToolID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: tool id is required", ErrInvalidTool)
	}
	return nil
}

// Status is the closed set of lifecycle states a Tool may be in
// (SPEC §8.4).
type Status string

const (
	// StatusActive marks a Tool as available for use.
	StatusActive Status = "ACTIVE"
	// StatusDeprecated marks a Tool as discouraged but still usable.
	StatusDeprecated Status = "DEPRECATED"
	// StatusDisabled marks a Tool as unavailable for use.
	StatusDisabled Status = "DISABLED"
)

// validStatuses is the closed Status set recognized by Validate.
var validStatuses = map[Status]struct{}{
	StatusActive:     {},
	StatusDeprecated: {},
	StatusDisabled:   {},
}

// Validate rejects an empty Status and any value outside the closed set.
func (s Status) Validate() error {
	if strings.TrimSpace(string(s)) == "" {
		return fmt.Errorf("%w: status is required", ErrInvalidTool)
	}
	if _, ok := validStatuses[s]; !ok {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidTool, string(s))
	}
	return nil
}

// ProvisionPolicy is the closed set of provision/update policies a Tool may
// carry (SPEC §8.4).
type ProvisionPolicy string

const (
	// ProvisionPolicyAuto allows AIDI to provision/update the tool without
	// human approval.
	ProvisionPolicyAuto ProvisionPolicy = "AUTO"
	// ProvisionPolicyRequestAdmin requires explicit admin approval before
	// provisioning/updating the tool.
	ProvisionPolicyRequestAdmin ProvisionPolicy = "REQUEST_ADMIN"
	// ProvisionPolicyDeny forbids provisioning/updating the tool.
	ProvisionPolicyDeny ProvisionPolicy = "DENY"
)

// validProvisionPolicies is the closed ProvisionPolicy set recognized by
// Validate.
var validProvisionPolicies = map[ProvisionPolicy]struct{}{
	ProvisionPolicyAuto:         {},
	ProvisionPolicyRequestAdmin: {},
	ProvisionPolicyDeny:         {},
}

// Validate rejects an empty ProvisionPolicy and any value outside the
// closed set.
func (p ProvisionPolicy) Validate() error {
	if strings.TrimSpace(string(p)) == "" {
		return fmt.Errorf("%w: provision policy is required", ErrInvalidTool)
	}
	if _, ok := validProvisionPolicies[p]; !ok {
		return fmt.Errorf("%w: unknown provision policy %q", ErrInvalidTool, string(p))
	}
	return nil
}

// ResourceNeeds describes the coarse resource footprint a Tool requires,
// shown by the Catalog alongside version/license/status (SPEC §8.4). All
// fields are optional/advisory: this domain contract does not enforce any
// particular unit or reserve/allocate resources itself (that is a later
// Resource Management slice, SPEC §10.4).
type ResourceNeeds struct {
	CPUCores int `json:"cpu_cores,omitempty"`
	MemoryMB int `json:"memory_mb,omitempty"`
	DiskMB   int `json:"disk_mb,omitempty"`
}

// Tool is a Software and Service Catalog entry (SPEC §8.4): a named,
// versioned, licensed piece of software AIDI may apply on engineering
// stages, together with its lifecycle Status, provisioning policy,
// advisory resource needs, and a list of other ToolIDs it depends on.
type Tool struct {
	ID              ToolID          `json:"id"`
	Name            string          `json:"name"`
	Version         string          `json:"version"`
	License         string          `json:"license"`
	Status          Status          `json:"status"`
	ProvisionPolicy ProvisionPolicy `json:"provision_policy"`
	ResourceNeeds   ResourceNeeds   `json:"resource_needs,omitempty"`
	Dependencies    []ToolID        `json:"dependencies,omitempty"`
}

// ErrInvalidTool is returned (wrapped) when a Tool fails validation: an
// empty ToolID/name/version/license, an unknown Status, or an unknown
// ProvisionPolicy.
var ErrInvalidTool = errors.New("toolregistry: invalid tool")

// ErrDuplicateTool is returned (wrapped) when Registry.Register is called
// with a ToolID that is already registered.
var ErrDuplicateTool = errors.New("toolregistry: duplicate tool id")

// Validate rejects a Tool with an empty ToolID, Name, Version, or License,
// an unknown Status, or an unknown ProvisionPolicy. It never panics.
func (t Tool) Validate() error {
	if err := t.ID.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTool)
	}
	if strings.TrimSpace(t.Version) == "" {
		return fmt.Errorf("%w: version is required", ErrInvalidTool)
	}
	if strings.TrimSpace(t.License) == "" {
		return fmt.Errorf("%w: license is required", ErrInvalidTool)
	}
	if err := t.Status.Validate(); err != nil {
		return err
	}
	if err := t.ProvisionPolicy.Validate(); err != nil {
		return err
	}
	return nil
}

// Registry is an in-memory, pure-function domain aggregate of Tool
// entries, keyed by ToolID. The zero value is not ready for use; construct
// one with NewRegistry. Registry holds no clock, global, or singleton
// state and performs no I/O: it is a plain in-memory map guarded only by
// the invariants enforced in Register.
type Registry struct {
	tools map[ToolID]Tool
}

// NewRegistry returns an empty, ready-to-use Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[ToolID]Tool)}
}

// Register validates tool and adds it to the Registry. It rejects an
// invalid Tool (see Tool.Validate) and a Tool whose ID is already
// registered (ErrDuplicateTool), never panics, and never mutates the
// Registry on error.
func (r *Registry) Register(tool Tool) error {
	if err := tool.Validate(); err != nil {
		return err
	}
	if r.tools == nil {
		r.tools = make(map[ToolID]Tool)
	}
	if _, exists := r.tools[tool.ID]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateTool, tool.ID)
	}
	r.tools[tool.ID] = tool
	return nil
}

// Get returns the Tool registered under id and true, or a zero Tool and
// false if no such Tool is registered.
func (r *Registry) Get(id ToolID) (Tool, bool) {
	if r == nil || r.tools == nil {
		return Tool{}, false
	}
	tool, ok := r.tools[id]
	return tool, ok
}

// List returns every registered Tool, ordered deterministically by ToolID.
// It returns an empty (non-nil) slice for an empty Registry.
func (r *Registry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}
