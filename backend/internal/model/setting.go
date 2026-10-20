package model

import "encoding/json"

// Setting is one stored key-value row of a settings group.
type Setting struct {
	Group    string
	Key      string
	Value    json.RawMessage
	IsSecret bool
}
