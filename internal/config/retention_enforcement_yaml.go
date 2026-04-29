package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// retentionEnforcementField decodes retention.enforcement from YAML.
// Use boolean true/false (recommended): true means the server rejects put_cell kinds that conflict
// with capture_mode; false means advisory only (same as legacy enforcement: off).
// For backward compatibility, strings "off" and "reject" are still accepted.
type retentionEnforcementField struct {
	present bool
	value   PolicyEnforcement
}

// UnmarshalYAML implements [yaml.Unmarshaler].
func (e *retentionEnforcementField) UnmarshalYAML(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	e.present = true
	var raw any
	if err := n.Decode(&raw); err != nil {
		return fmt.Errorf("mosaic config: retention enforcement: %w", err)
	}
	switch v := raw.(type) {
	case bool:
		if v {
			e.value = PolicyEnforcementReject
		} else {
			e.value = PolicyEnforcementOff
		}
		return nil
	case string:
		s := strings.TrimSpace(strings.ToLower(v))
		switch s {
		case "off":
			e.value = PolicyEnforcementOff
		case "reject":
			e.value = PolicyEnforcementReject
		default:
			return fmt.Errorf("mosaic config: enforcement must be true or false (legacy: off or reject), got %q", v)
		}
		return nil
	default:
		return fmt.Errorf("mosaic config: enforcement must be boolean true or false, got %T", raw)
	}
}

func (e retentionEnforcementField) resolved(defaultWhenAbsent PolicyEnforcement) PolicyEnforcement {
	if !e.present {
		return defaultWhenAbsent
	}
	return e.value
}
