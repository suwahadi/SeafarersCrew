package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// RevalidateJob is the job kind that notifies the website about changed content.
const RevalidateJob = "site.revalidate"

// TagBumper invalidates public API cache entries by tag.
type TagBumper interface {
	BumpTags(ctx context.Context, tags []string) error
}

// Revalidator tells the website to drop cached data.
type Revalidator interface {
	Enabled() bool
	Revalidate(ctx context.Context, tags []string) error
}

// SitePublisher propagates published content changes to the API cache and the website.
type SitePublisher struct {
	cache       TagBumper
	jobs        Enqueuer
	revalidator Revalidator
}

// NewSitePublisher builds a SitePublisher.
func NewSitePublisher(cache TagBumper, jobs Enqueuer, revalidator Revalidator) *SitePublisher {
	return &SitePublisher{cache: cache, jobs: jobs, revalidator: revalidator}
}

type revalidatePayload struct {
	Tags []string `json:"tags"`
}

// Changed invalidates the API cache for tags and queues website revalidation.
func (p *SitePublisher) Changed(ctx context.Context, tags ...string) error {
	tags = uniqueTags(tags)
	if len(tags) == 0 {
		return nil
	}
	if err := p.cache.BumpTags(ctx, tags); err != nil {
		return err
	}
	if !p.revalidator.Enabled() {
		return nil
	}
	if _, err := p.jobs.Enqueue(ctx, RevalidateJob, revalidatePayload{Tags: tags}); err != nil {
		return fmt.Errorf("queue website revalidation: %w", err)
	}
	return nil
}

// Handle runs a queued revalidation.
func (p *SitePublisher) Handle(ctx context.Context, j model.Job) (any, error) {
	var payload revalidatePayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return nil, job.Permanent{Err: fmt.Errorf("decode revalidate payload: %w", err)}
	}
	if err := p.revalidator.Revalidate(ctx, payload.Tags); err != nil {
		return nil, err
	}
	return payload, nil
}

func uniqueTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}
