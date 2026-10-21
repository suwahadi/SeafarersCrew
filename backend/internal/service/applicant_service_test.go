package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type moveRecord struct {
	id       uuid.UUID
	from, to string
	note     string
	admin    uuid.UUID
}

type memApplicants struct {
	rows       map[uuid.UUID]model.Applicant
	moves      []moveRecord
	staleStage bool
	lastFilter model.ApplicantFilter
	counts     map[string]int
}

func (m *memApplicants) GetByPublicID(_ context.Context, publicID string) (model.Applicant, error) {
	for _, a := range m.rows {
		if a.PublicID == publicID {
			return a, nil
		}
	}
	return model.Applicant{}, repository.ErrNotFound
}

func (m *memApplicants) List(_ context.Context, f model.ApplicantFilter, _, _ int) (repository.Page[model.Applicant], error) {
	m.lastFilter = f
	return repository.Page[model.Applicant]{}, nil
}

func (m *memApplicants) StageCounts(_ context.Context, f model.ApplicantFilter) (map[string]int, error) {
	m.lastFilter = f
	return m.counts, nil
}

func (m *memApplicants) ChangeStage(_ context.Context, id uuid.UUID, from, to, note string, admin uuid.UUID) error {
	if m.staleStage {
		return repository.ErrStateChanged
	}
	a := m.rows[id]
	a.Stage = to
	m.rows[id] = a
	m.moves = append(m.moves, moveRecord{id: id, from: from, to: to, note: note, admin: admin})
	return nil
}

func (m *memApplicants) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.rows[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.rows, id)
	return nil
}

func newApplicantFixture() (*ApplicantService, *memApplicants, model.Applicant) {
	a := model.Applicant{ID: uuid.New(), PublicID: "APP222222", Stage: model.StageNew}
	store := &memApplicants{rows: map[uuid.UUID]model.Applicant{a.ID: a}}
	now := func() time.Time { return time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC) }
	return NewApplicantService(store, staticSettings{"general": {"timezone": "Asia/Jakarta"}}, now), store, a
}

func TestChangeStageAllowsAnyMoveExceptStayingPut(t *testing.T) {
	admin := uuid.New()
	for _, from := range model.ApplicantStages {
		for _, to := range model.ApplicantStages {
			t.Run(from+" to "+to, func(t *testing.T) {
				svc, store, a := newApplicantFixture()
				a.Stage = from
				store.rows[a.ID] = a

				got, err := svc.ChangeStage(context.Background(), a.PublicID, admin, to, "reviewed")

				if from == to {
					requireAppError(t, err, 409, "stage_unchanged")
					assert.Empty(t, store.moves)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, to, got.Stage)
				require.Len(t, store.moves, 1)
				assert.Equal(t, moveRecord{id: a.ID, from: from, to: to, note: "reviewed", admin: admin}, store.moves[0])
			})
		}
	}
}

func TestChangeStageRequiresNoteWhenRejecting(t *testing.T) {
	svc, store, a := newApplicantFixture()

	_, err := svc.ChangeStage(context.Background(), a.PublicID, uuid.New(), model.StageRejected, "   ")

	fields := appErrorFields(t, err, 422, "validation_failed")
	assert.Contains(t, fields, "note")
	assert.Empty(t, store.moves)

	_, err = svc.ChangeStage(context.Background(), a.PublicID, uuid.New(), model.StageRejected, " Does not hold a valid COC ")
	require.NoError(t, err)
	assert.Equal(t, "Does not hold a valid COC", store.moves[0].note, "notes are trimmed")
}

func TestChangeStageAllowsEmptyNoteForOtherStages(t *testing.T) {
	svc, store, a := newApplicantFixture()

	_, err := svc.ChangeStage(context.Background(), a.PublicID, uuid.New(), model.StageScreening, "")

	require.NoError(t, err)
	assert.Equal(t, "", store.moves[0].note)
}

func TestChangeStageValidatesStageAndNoteLength(t *testing.T) {
	tests := []struct {
		name, stage, note, field string
	}{
		{"unknown stage", "hired", "", "stage"},
		{"empty stage", "", "", "stage"},
		{"note too long", model.StageInterview, strings.Repeat("é", StageNoteMaxLength+1), "note"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, store, a := newApplicantFixture()

			_, err := svc.ChangeStage(context.Background(), a.PublicID, uuid.New(), tc.stage, tc.note)

			fields := appErrorFields(t, err, 422, "validation_failed")
			assert.Contains(t, fields, tc.field)
			assert.Empty(t, store.moves)
		})
	}
}

func TestChangeStageReportsConcurrentMove(t *testing.T) {
	svc, store, a := newApplicantFixture()
	store.staleStage = true

	_, err := svc.ChangeStage(context.Background(), a.PublicID, uuid.New(), model.StageScreening, "")

	requireAppError(t, err, 409, "stage_changed")
}

func TestApplicantLookupWithUnknownOrMalformedIDIsNotFound(t *testing.T) {
	svc, _, _ := newApplicantFixture()
	for _, id := range []string{"APP999999", "bad", ""} {
		_, err := svc.Get(context.Background(), id)
		var e *apperr.Error
		require.ErrorAs(t, err, &e)
		assert.Equal(t, 404, e.Status)
	}
	_, err := svc.ChangeStage(context.Background(), "APP999999", uuid.New(), model.StageScreening, "")
	assert.Error(t, err)
}

func TestApplicantGetAcceptsLowercasePublicID(t *testing.T) {
	svc, _, a := newApplicantFixture()

	got, err := svc.Get(context.Background(), strings.ToLower(a.PublicID))

	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)
}

func TestApplicantListTurnsDatesIntoInstantsInTheSettingsTimezone(t *testing.T) {
	svc, store, _ := newApplicantFixture()
	from, to := day(2026, 10, 1), day(2026, 10, 31)

	_, err := svc.List(context.Background(), model.ApplicantFilter{DateFrom: from, DateTo: to}, 1, 20)

	require.NoError(t, err)
	require.NotNil(t, store.lastFilter.CreatedFrom)
	require.NotNil(t, store.lastFilter.CreatedBefore)
	assert.Equal(t, time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC), *store.lastFilter.CreatedFrom, "Jakarta midnight of the first day")
	assert.Equal(t, time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC), *store.lastFilter.CreatedBefore, "the end date is inclusive")
}

func TestStageCountsIncludeEveryStage(t *testing.T) {
	svc, store, _ := newApplicantFixture()
	store.counts = map[string]int{model.StageNew: 3, model.StageRejected: 1}

	counts, err := svc.StageCounts(context.Background(), model.ApplicantFilter{})

	require.NoError(t, err)
	assert.Equal(t, map[string]int{"new": 3, "screening": 0, "interview": 0, "accepted": 0, "rejected": 1}, counts)
}

func TestDeleteApplicantIsSoft(t *testing.T) {
	svc, store, a := newApplicantFixture()

	require.NoError(t, svc.Delete(context.Background(), a.PublicID))

	assert.Empty(t, store.rows)
	assert.Error(t, svc.Delete(context.Background(), a.PublicID))
}
