package seeders

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/config"
)

func TestCheckDemoAllowed(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		force   bool
		wantErr bool
	}{
		{"local without force", config.EnvLocal, false, false},
		{"staging without force", config.EnvStaging, false, false},
		{"production without force", config.EnvProduction, false, true},
		{"production with force", config.EnvProduction, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckDemoAllowed(tc.env, tc.force)
			if tc.wantErr {
				assert.ErrorIs(t, err, ErrDemoInProduction)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestStepRNGIsDeterministicPerStepName(t *testing.T) {
	assert.Equal(t, stepRNG("crew").Uint64(), stepRNG("crew").Uint64())
	assert.NotEqual(t, stepRNG("crew").Uint64(), stepRNG("cms").Uint64())
}

func TestClientURLBuildsAnExampleDomainFromTheName(t *testing.T) {
	assert.Equal(t, "https://samudera-nusantara-offshore.example.com", clientURL("Samudera Nusantara Offshore"))
}
