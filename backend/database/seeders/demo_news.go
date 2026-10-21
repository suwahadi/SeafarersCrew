package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type newsSeed struct {
	title, excerpt, body string
	daysAgo              int
}

var demoNews = []newsSeed{
	{
		"Crew change planning: why early relief matters",
		"A relief that is lined up weeks ahead keeps vessels safe and seafarers rested. Here is how we plan crew changes.",
		`<p>A crew change is smoother when the relief is named early. Our rotation planning shows who is on board each vessel and when the next relief is due, so a qualified replacement can be confirmed well before the contract ends.</p><h2>What we check</h2><ul><li>The reliever is available for the whole contract period.</li><li>Every mandatory document and certificate stays valid until planned sign-off.</li><li>Travel, visa and joining port are arranged with the principal.</li></ul><p>Seafarers who keep their documents up to date make this planning faster for everyone.</p>`,
		4,
	},
	{
		"Renew your seaman book and medical certificate early",
		"Expired documents are the most common reason a joining is delayed. Check your expiry dates today.",
		`<p>A passport, seaman book or medical check-up that expires during a contract can delay a joining or force an early sign-off. We recommend starting renewals at least 90 days before the expiry date.</p><h2>Our reminder colours</h2><ul><li><strong>Yellow:</strong> 61 to 90 days left</li><li><strong>Orange:</strong> 31 to 60 days left</li><li><strong>Red:</strong> 30 days or less</li></ul><p>Send your renewed copies to the crewing team as soon as you receive them.</p>`,
		14,
	},
	{
		"New online application portal for seafarers",
		"Applying for a vacancy is now fully online, free of charge, and you can attach your CV and documents in one step.",
		`<p>Seafarers can now browse open vacancies and apply from any device. Choose a vacancy, fill in the short form, attach your CV and supporting documents, and submit.</p><p>Every application moves through clear stages, from New to Accepted or Rejected, and our crewing team contacts you by email or phone if your profile matches. We never charge applicants any fee.</p>`,
		40,
	},
	{
		"Safety first: refresher courses for deck and engine crew",
		"Basic Safety Training and Advanced Fire Fighting certificates are valid for five years. Plan your refresher in time.",
		`<p>STCW certificates such as Basic Safety Training (VI/1), Advanced Fire Fighting (VI/3) and Medical First Aid (VI/4) need refreshing before they expire. Booking a course a few months ahead avoids gaps between contracts.</p><p>Our crewing team can share a list of approved training centres on request.</p>`,
		95,
	},
	{
		"Offshore support vessels: what principals ask of their crew",
		"We spoke with operators about the qualities they look for in masters, engineers and ratings on offshore support vessels.",
		`<p>Principals consistently ask for the same things: valid certificates, sound sea experience on similar vessels, offshore survival training such as BOSIET and HUET, and a professional attitude on board.</p><p>Keeping your sea service record complete and accurate helps us match you with the right vessel.</p>`,
		210,
	},
	{
		"Upcoming: crew welfare day and open house",
		"Join our crewing team for a welfare day with document clinics and career advice for seafarers and their families.",
		`<p>We will hold a welfare day with document clinics, career advice and an open Q&amp;A with our crewing team. Seafarers and their families are welcome. Details of the venue and programme will be announced here.</p>`,
		-21,
	},
}

func seedDemoNews(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	images, err := d.loadImages(ctx)
	if err != nil {
		return err
	}
	for i, n := range demoNews {
		published := d.today.AddDate(0, 0, -n.daysAgo).Add(9 * time.Hour)
		thumbnail := images.news[i]
		if _, err := d.News.Create(ctx, newsModel(n, published, &thumbnail)); err != nil {
			return fmt.Errorf("create news article %d: %w", i+1, err)
		}
	}
	return nil
}

func newsModel(n newsSeed, published time.Time, thumbnail *uuid.UUID) model.News {
	return model.News{
		Title: n.title, Excerpt: n.excerpt, BodyHTML: n.body, ThumbnailFileID: thumbnail, PublishedAt: &published,
		Status: model.StatusPublished, SEOTitle: n.title, MetaDescription: n.excerpt,
	}
}

func seedDemoAboutPage(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	images, err := d.loadImages(ctx)
	if err != nil {
		return err
	}
	var id uuid.UUID
	err = d.Pool.QueryRow(ctx, `SELECT id FROM pages WHERE slug = $1 AND deleted_at IS NULL`, AboutUsSlug).Scan(&id)
	if err != nil {
		return fmt.Errorf("find about page: %w", err)
	}
	detail, err := d.Pages.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("load about page: %w", err)
	}
	page := detail.Page
	for _, b := range page.Blocks {
		if b.Type == model.BlockImageText {
			return nil
		}
	}
	about := images.about
	page.Blocks = append([]model.Block{{ID: "story-image", Type: model.BlockImageText, Settings: &model.ImageTextSettings{
		ImageFileID:   &about,
		Heading:       "People who keep ships moving",
		TextHTML:      "<p>Behind every voyage is a crew that is qualified, rested and well prepared. We look after the people, the paperwork and the planning so that principals can focus on their vessels.</p>",
		ButtonLabel:   "See current vacancies",
		ButtonLink:    "/career",
		ImagePosition: "right",
	}}}, page.Blocks...)
	page.OGImageFileID = &about
	if _, err := d.Pages.Update(ctx, id, page); err != nil {
		return fmt.Errorf("add image to about page: %w", err)
	}
	return nil
}
