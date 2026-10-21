package repository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

var companyPageSlugs = []string{"about-us", "services", "terms-and-conditions", "privacy-policy", "faq"}

func seedReferenceFresh(t *testing.T) {
	t.Helper()
	truncate(t, "pages", "menu_items", "languages", "document_templates", "redirects")
	require.NoError(t, seeders.Reference(context.Background(), testDB, noSettings{}, nil))
}

func pageBySlug(t *testing.T, slug string) model.Page {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT id FROM pages WHERE slug = $1 AND deleted_at IS NULL`, slug).Scan(&id))
	p, err := repository.NewPageRepository(testDB).Get(context.Background(), id)
	require.NoError(t, err)
	return p
}

func htmlOf(p model.Page) string {
	var b strings.Builder
	for _, blk := range p.Blocks {
		switch v := blk.Settings.(type) {
		case *model.RichTextSettings:
			b.WriteString(v.HTML)
		case *model.FAQSettings:
			for _, it := range v.Items {
				b.WriteString(it.Question + it.AnswerHTML)
			}
		}
	}
	return b.String()
}

func blockTypes(p model.Page) []string {
	types := make([]string, len(p.Blocks))
	for i, b := range p.Blocks {
		types[i] = b.Type
	}
	return types
}

func TestReferenceSeedCreatesPublishedCompanyPagesWithValidBlocks(t *testing.T) {
	seedReferenceFresh(t)

	for _, slug := range companyPageSlugs {
		p := pageBySlug(t, slug)
		assert.Equal(t, model.StatusPublished, p.Status, slug)
		assert.NotNil(t, p.PublishedAt, slug)
		assert.False(t, p.IsSystem, slug)
		assert.NotEmpty(t, p.SEOTitle, slug)
		assert.NotEmpty(t, p.MetaDescription, slug)
		assert.NotEmpty(t, p.Blocks, slug)
	}

	about := pageBySlug(t, "about-us")
	assert.Equal(t, []string{model.BlockRichText, model.BlockRichText, model.BlockStats, model.BlockRichText, model.BlockCardGrid}, blockTypes(about))
	assert.True(t, about.InHeaderMenu)
	assert.False(t, about.InFooterMenu)

	services := pageBySlug(t, "services")
	assert.Equal(t, []string{model.BlockCardGrid, model.BlockCTA}, blockTypes(services))
	cta := services.Blocks[1].Settings.(*model.CTASettings)
	assert.Equal(t, "/career", cta.ButtonLink)
	assert.True(t, services.InHeaderMenu)

	faq := pageBySlug(t, "faq")
	require.Len(t, faq.Blocks, 1)
	assert.GreaterOrEqual(t, len(faq.Blocks[0].Settings.(*model.FAQSettings).Items), 8)
	assert.False(t, faq.InHeaderMenu || faq.InFooterMenu)
}

func TestReferenceSeedLegalPagesCoverTheRequiredTopics(t *testing.T) {
	seedReferenceFresh(t)

	terms := pageBySlug(t, "terms-and-conditions")
	assert.True(t, terms.InFooterMenu)
	termsText := htmlOf(terms)
	for _, topic := range []string{"Use of this website", "Career applications", "accurate", "does not charge applicants any fee", "Intellectual property",
		"Links to other websites", "Limitation of liability", "Republic of Indonesia", "Changes to these terms", "Contact"} {
		assert.Contains(t, termsText, topic)
	}

	privacy := pageBySlug(t, "privacy-policy")
	assert.True(t, privacy.InFooterMenu)
	privacyText := htmlOf(privacy)
	for _, topic := range []string{"data controller", "Contact form", "Career application form", "Recruitment", "Communication", "consent", "24 months",
		"principals", "Storage and security", "No. 27 of 2022", "UU PDP", "Language preference", "Turnstile", "Contact for privacy requests"} {
		assert.Contains(t, privacyText, topic)
	}
}

func TestReferenceSeedSecondRunChangesNothing(t *testing.T) {
	seedReferenceFresh(t)
	ctx := context.Background()
	snapshot := func() []string {
		rows, err := testDB.Query(ctx, `SELECT slug || '|' || title || '|' || blocks::text || '|' || updated_at::text FROM pages ORDER BY slug`)
		require.NoError(t, err)
		out, err := pgx.CollectRows(rows, pgx.RowTo[string])
		require.NoError(t, err)
		return out
	}
	menus := func() []string {
		rows, err := testDB.Query(ctx, `SELECT menu || '|' || label || '|' || sort_order FROM menu_items ORDER BY menu, sort_order, label`)
		require.NoError(t, err)
		out, err := pgx.CollectRows(rows, pgx.RowTo[string])
		require.NoError(t, err)
		return out
	}
	pagesBefore, menusBefore := snapshot(), menus()

	require.NoError(t, seeders.Reference(ctx, testDB, noSettings{}, nil))

	assert.Equal(t, pagesBefore, snapshot())
	assert.Equal(t, menusBefore, menus())
	assert.Len(t, pagesBefore, 10)
}

func TestReferenceSeedKeepsAdminEditsToCompanyPages(t *testing.T) {
	seedReferenceFresh(t)
	ctx := context.Background()
	_, err := testDB.Exec(ctx, `UPDATE pages SET title = 'Who We Are', blocks = '[]'::jsonb, status = 'draft' WHERE slug = 'about-us'`)
	require.NoError(t, err)
	_, err = testDB.Exec(ctx, `UPDATE pages SET deleted_at = now() WHERE slug = 'faq'`)
	require.NoError(t, err)

	require.NoError(t, seeders.Reference(ctx, testDB, noSettings{}, nil))

	var title, status string
	var blocks int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT title, status, jsonb_array_length(blocks) FROM pages WHERE slug = 'about-us'`).Scan(&title, &status, &blocks))
	assert.Equal(t, "Who We Are", title)
	assert.Equal(t, model.StatusDraft, status)
	assert.Zero(t, blocks)
	var live int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM pages WHERE slug = 'faq' AND deleted_at IS NULL`).Scan(&live))
	assert.Zero(t, live, "a page the admin deleted is not brought back")
}

func TestReferenceSeedAppendsCompanyPagesToTheDefaultMenusOnce(t *testing.T) {
	seedReferenceFresh(t)
	require.NoError(t, seeders.Reference(context.Background(), testDB, noSettings{}, nil))

	labels := func(menu string) []string {
		var out []string
		for _, m := range menuItemsOf(t, menu) {
			out = append(out, m.Label)
		}
		return out
	}
	assert.Equal(t, []string{"Home", "News", "Career", "Gallery", "Contact", "About Us", "Services"}, labels(model.MenuHeader))
	assert.Equal(t, []string{"News", "Career", "Contact", "Terms and Conditions", "Privacy Policy"}, labels(model.MenuFooter))
}
