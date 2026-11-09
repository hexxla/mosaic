package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// retentionEnforcementField accepts only YAML boolean true/false.
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
	if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" {
		return fmt.Errorf("mosaic config: retention.enforcement must be boolean true or false")
	}
	var enabled bool
	if err := n.Decode(&enabled); err != nil {
		return fmt.Errorf("mosaic config: retention enforcement: %w", err)
	}
	e.value = PolicyEnforcementOff
	if enabled {
		e.value = PolicyEnforcementReject
	}
	return nil
}

func (e retentionEnforcementField) resolved(defaultWhenAbsent PolicyEnforcement) PolicyEnforcement {
	if !e.present {
		return defaultWhenAbsent
	}
	return e.value
}
