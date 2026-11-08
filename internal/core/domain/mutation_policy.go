package domain

import "errors"

// MutationPolicy holds application decisions derived from operator configuration.
type MutationPolicy struct {
	UserMessageDenied       error
	AssistantResponseDenied error
	AllowDeleteCell         bool
}

// PutCellDenied returns the configured denial for a conversational cell kind.
func (p MutationPolicy) PutCellDenied(kind CellPutKind) error {
	switch kind {
	case CellPutKindUserMessage:
		return p.UserMessageDenied
	case CellPutKindAssistantResponse:
		return p.AssistantResponseDenied
	default:
		return nil
	}
}

// DeleteCellDenied reports whether the operator enabled cell deletion.
func (p MutationPolicy) DeleteCellDenied() error {
	if p.AllowDeleteCell {
		return nil
	}
	return errors.New("mosaic config: delete_cell not allowed (allow_delete_cell is false)")
}
