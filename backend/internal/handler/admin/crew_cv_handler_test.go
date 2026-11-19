package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

type stubCVImporter struct {
	filename string
	data     []byte
	commit   *service.CVCommitInput
}

func (s *stubCVImporter) Validate(_ context.Context, filename string, data []byte) (service.CVPreview, error) {
	s.filename, s.data = filename, data
	return service.CVPreview{Token: "tok", FullName: "BUDI SANTOSO"}, nil
}

func (s *stubCVImporter) Commit(_ context.Context, in service.CVCommitInput) (service.CVCommitResult, error) {
	s.commit = &in
	return service.CVCommitResult{CrewID: "CVCREW002", FullName: "BUDI SANTOSO"}, nil
}

func TestCrewCVValidateRejectsAnythingButXLSX(t *testing.T) {
	for _, name := range []string{"cv.xls", "cv.csv", "cv"} {
		t.Run(name, func(t *testing.T) {
			stub := &stubCVImporter{}
			body, ct := multipartBody(t, nil, name, []byte("data"))
			req := httptest.NewRequest(http.MethodPost, "/crews/cv-import/validate", body)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			NewCrewCVHandler(stub, &stubFiles{}).Validate(rec, req)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			assert.Contains(t, rec.Body.String(), "Save the CV as .xlsx and upload it again.")
			assert.Nil(t, stub.data, "the service is not called")
		})
	}
}

func TestCrewCVValidatePassesTheWorkbook(t *testing.T) {
	stub := &stubCVImporter{}
	body, ct := multipartBody(t, nil, "CV RVOS 2024.XLSX", []byte("xlsx-bytes"))
	req := httptest.NewRequest(http.MethodPost, "/crews/cv-import/validate", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	NewCrewCVHandler(stub, &stubFiles{}).Validate(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "CV RVOS 2024.XLSX", stub.filename)
	assert.Equal(t, "xlsx-bytes", string(stub.data))
	assert.Contains(t, rec.Body.String(), `"token":"tok"`)
}

func TestCrewCVCommitDecodesDecisions(t *testing.T) {
	stub := &stubCVImporter{}
	req := httptest.NewRequest(http.MethodPost, "/crews/cv-import/commit",
		strings.NewReader(`{"token":"tok","decisions":{"rank:XX":{"action":"match","target_id":"00000000-0000-0000-0000-000000000020"}}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	NewCrewCVHandler(stub, &stubFiles{}).Commit(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.commit)
	assert.Equal(t, "tok", stub.commit.Token)
	assert.Equal(t, service.CVDecision{Action: "match", TargetID: "00000000-0000-0000-0000-000000000020"}, stub.commit.Decisions["rank:XX"])
	var out struct {
		Data service.CVCommitResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, "CVCREW002", out.Data.CrewID)
}
