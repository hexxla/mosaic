package config

import (
	"fmt"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// MosaicConfigVersion is the top-level config file schema version.
const MosaicConfigVersion = 1

// CaptureMode describes what classes of turn data the product intends to persist into Hexxla.
// It is advisory for agents unless [RetentionPolicy.Enforcement] is "reject".
type CaptureMode string

const (
	// CaptureModeLLMCurates means the model chooses what to store.
	CaptureModeLLMCurates CaptureMode = "llm_curates"
	// CaptureModeSaveAllTurns allows both user and assistant turn kinds in scope.
	CaptureModeSaveAllTurns CaptureMode = "save_all_turns"
	// CaptureModeUserOnly scopes turn persistence to user messages.
	CaptureModeUserOnly CaptureMode = "user_only"
	// CaptureModeAssistantOnly scopes turn persistence to assistant messages.
	CaptureModeAssistantOnly CaptureMode = "assistant_only"
	// CaptureModeNone means no chat turn cells in scope for retention.
	CaptureModeNone CaptureMode = "none"
)

// PolicyEnforcement controls whether the MCP server blocks puts that violate [CaptureMode].
// Config YAML uses booleans: false → off, true → reject. JSON/MCP responses still use the strings "off" and "reject".
type PolicyEnforcement string

const (
	// PolicyEnforcementOff does not block puts server-side (YAML: enforcement: false).
	PolicyEnforcementOff PolicyEnforcement = "off"
	// PolicyEnforcementReject blocks disallowed put_cell kinds (YAML: enforcement: true).
	PolicyEnforcementReject PolicyEnforcement = "reject"
)

// RetentionPolicy holds capture/enforcement intent for turn-related puts (YAML key: retention).
type RetentionPolicy struct {
	Version     int               `json:"version" yaml:"version"`
	CaptureMode CaptureMode       `json:"capture_mode" yaml:"capture_mode"`
	Enforcement PolicyEnforcement `json:"enforcement" yaml:"enforcement"`
	Notes       string            `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// DefaultRetentionPolicy is used when no config file or no retention section is present.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		Version:     MosaicConfigVersion,
		CaptureMode: CaptureModeLLMCurates,
		Enforcement: PolicyEnforcementOff,
	}
}

// CheckPutCell returns nil if the put is allowed, or an error when enforcement rejects the kind.
func (p *RetentionPolicy) CheckPutCell(kind domain.CellPutKind) error {
	if p == nil || p.Enforcement != PolicyEnforcementReject {
		return nil
	}
	k := kind
	if k == "" {
		k = domain.CellPutKindFact
	}
	switch p.CaptureMode {
	case CaptureModeLLMCurates, CaptureModeSaveAllTurns:
		return nil
	case CaptureModeNone:
		if k == domain.CellPutKindUserMessage || k == domain.CellPutKindAssistantResponse {
			return fmt.Errorf("retention policy: capture_mode %q rejects chat turns", p.CaptureMode)
		}
	case CaptureModeUserOnly:
		if k == domain.CellPutKindAssistantResponse {
			return fmt.Errorf("retention policy: capture_mode %q rejects assistant_message puts", p.CaptureMode)
		}
	case CaptureModeAssistantOnly:
		if k == domain.CellPutKindUserMessage {
			return fmt.Errorf("retention policy: capture_mode %q rejects user_message puts", p.CaptureMode)
		}
	}
	return nil
}
