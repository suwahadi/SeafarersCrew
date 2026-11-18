package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func TestNormalizeEmployeeNo(t *testing.T) {
	for in, want := range map[string]string{" 900101 ": "900101", "00412": "412", "ab-7": "AB-7", "": ""} {
		assert.Equal(t, want, model.NormalizeEmployeeNo(in), in)
	}
}
