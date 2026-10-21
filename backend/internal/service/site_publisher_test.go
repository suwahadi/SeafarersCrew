package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type fakeBumper struct{ tags [][]string }

func (f *fakeBumper) BumpTags(_ context.Context, tags []string) error {
	f.tags = append(f.tags, tags)
	return nil
}

type fakeQueue struct {
	kinds    []string
	payloads []any
}

func (f *fakeQueue) Enqueue(_ context.Context, kind string, payload any) (model.Job, error) {
	f.kinds = append(f.kinds, kind)
	f.payloads = append(f.payloads, payload)
	return model.Job{Kind: kind}, nil
}

type fakeRevalidator struct {
	enabled bool
	got     []string
	err     error
}

func (f *fakeRevalidator) Enabled() bool { return f.enabled }

func (f *fakeRevalidator) Revalidate(_ context.Context, tags []string) error {
	f.got = tags
	return f.err
}

func TestSitePublisherBumpsCacheAndQueuesRevalidationOnce(t *testing.T) {
	cache, queue := &fakeBumper{}, &fakeQueue{}
	p := NewSitePublisher(cache, queue, &fakeRevalidator{enabled: true})

	require.NoError(t, p.Changed(context.Background(), model.TagNews, model.TagHome, model.TagNews, ""))

	assert.Equal(t, [][]string{{"news", "home"}}, cache.tags)
	assert.Equal(t, []string{RevalidateJob}, queue.kinds)
	assert.Equal(t, revalidatePayload{Tags: []string{"news", "home"}}, queue.payloads[0])
}

func TestSitePublisherSkipsRevalidationWithoutWebsite(t *testing.T) {
	cache, queue := &fakeBumper{}, &fakeQueue{}
	p := NewSitePublisher(cache, queue, &fakeRevalidator{})
	require.NoError(t, p.Changed(context.Background(), model.TagSite))
	assert.Len(t, cache.tags, 1)
	assert.Empty(t, queue.kinds)
}

func TestSitePublisherIgnoresEmptyChanges(t *testing.T) {
	cache := &fakeBumper{}
	require.NoError(t, NewSitePublisher(cache, &fakeQueue{}, &fakeRevalidator{enabled: true}).Changed(context.Background()))
	assert.Empty(t, cache.tags)
}

func TestSitePublisherJobForwardsTagsAndRetriesOnFailure(t *testing.T) {
	rv := &fakeRevalidator{enabled: true}
	p := NewSitePublisher(&fakeBumper{}, &fakeQueue{}, rv)
	payload, _ := json.Marshal(revalidatePayload{Tags: []string{"pages"}})

	_, err := p.Handle(context.Background(), model.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, []string{"pages"}, rv.got)

	rv.err = errors.New("down")
	_, err = p.Handle(context.Background(), model.Job{Payload: payload})
	var perm job.Permanent
	assert.Error(t, err)
	assert.False(t, errors.As(err, &perm))

	_, err = p.Handle(context.Background(), model.Job{Payload: []byte("{")})
	assert.True(t, errors.As(err, &perm))
}
