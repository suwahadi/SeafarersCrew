package service

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type siteRecorder struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *siteRecorder) Changed(_ context.Context, tags ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, slices.Clone(tags))
	return nil
}

func (r *siteRecorder) last() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return nil
	}
	return r.calls[len(r.calls)-1]
}

func (r *siteRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

type fakeFileLookup struct {
	files map[uuid.UUID]model.PublicFileInfo
}

func newFakeFiles() *fakeFileLookup {
	return &fakeFileLookup{files: map[uuid.UUID]model.PublicFileInfo{}}
}

func (f *fakeFileLookup) add(mime, name string) uuid.UUID {
	id := uuid.New()
	f.files[id] = model.PublicFileInfo{ID: id, MimeType: mime, OriginalName: name}
	return id
}

func (f *fakeFileLookup) PublicFiles(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]model.PublicFileInfo, error) {
	out := map[uuid.UUID]model.PublicFileInfo{}
	for _, id := range ids {
		if info, ok := f.files[id]; ok {
			out[id] = info
		}
	}
	return out, nil
}

func fixedNow() time.Time { return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC) }
