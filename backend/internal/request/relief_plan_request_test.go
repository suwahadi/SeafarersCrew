package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReliefPlanValidateRequiresRelieverAndDateUnlessCancelled(t *testing.T) {
	tests := []struct {
		name       string
		body       ReliefPlan
		wantFields []string
	}{
		{"planned plan needs reliever and date", ReliefPlan{Status: "planned"}, []string{"reliever_crew_id", "planned_relief_date"}},
		{"cancelled plan needs neither", ReliefPlan{Status: "cancelled", Notes: "reliever withdrew"}, nil},
		{"confirmed plan with both is valid", ReliefPlan{Status: "confirmed", RelieverCrewID: "abcdefghj", PlannedReliefDate: "2026-12-01"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.body.Validate()
			assert.Len(t, got, len(tt.wantFields))
			for _, f := range tt.wantFields {
				assert.Contains(t, got, f)
			}
		})
	}
}
