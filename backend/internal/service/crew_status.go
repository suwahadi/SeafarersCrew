package service

import (
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// System crew status codes (PRD 11.2).
const (
	StatusAvailable    = "available"
	StatusOnBoard      = "on_board"
	StatusSignOff      = "sign_off"
	StatusMedicalLeave = "medical_leave"
	StatusBlacklisted  = "blacklisted"
)

// Manual status actions the admin may take.
const (
	ActionMedicalLeave = "medical_leave"
	ActionReady        = "ready"
	ActionBlacklist    = "blacklist"
)

// StatusChange is the result of a manual status action.
type StatusChange struct {
	Code   string
	Reason string
}

// ManualStatusChange validates a manual action and returns the target status (PRD 11.2).
func ManualStatusChange(action, reason string, hasActiveContract bool) (StatusChange, error) {
	reason = strings.TrimSpace(reason)
	if hasActiveContract {
		return StatusChange{}, apperr.Conflict("crew_on_board", "The crew member has an active contract; complete or terminate it first.")
	}
	switch action {
	case ActionMedicalLeave:
		return StatusChange{Code: StatusMedicalLeave, Reason: reason}, nil
	case ActionReady:
		return StatusChange{Code: StatusAvailable}, nil
	case ActionBlacklist:
		if reason == "" {
			return StatusChange{}, apperr.Validation(map[string]string{"reason": "A reason is required to blacklist a crew member."})
		}
		return StatusChange{Code: StatusBlacklisted, Reason: reason}, nil
	default:
		return StatusChange{}, apperr.Validation(map[string]string{"action": "The action must be medical_leave, ready or blacklist."})
	}
}

// CanBeAssigned reports whether a crew status allows a new Active contract or reliever plan.
func CanBeAssigned(statusCode string) bool {
	return statusCode == StatusAvailable
}
