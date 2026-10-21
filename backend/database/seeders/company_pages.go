package seeders

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

type noSite struct{}

func (noSite) Changed(context.Context, ...string) error { return nil }

type companyDetails struct {
	name, legalName, email, address string
}

func defaultCompanyDetails() companyDetails {
	return companyDetails{
		name:      settingDefault("general", "site_name"),
		legalName: settingDefault("company", "legal_name"),
		email:     settingDefault("company", "email"),
		address:   settingDefault("company", "address"),
	}
}

func settingDefault(group, key string) string {
	g, ok := service.FindSettingGroup(group)
	if !ok {
		return ""
	}
	f, ok := g.Field(key)
	if !ok {
		return ""
	}
	s, _ := f.Default.(string)
	return s
}

func (c companyDetails) fill(text string) string {
	return strings.NewReplacer(
		"{{name}}", html.EscapeString(c.name),
		"{{legal_name}}", html.EscapeString(c.legalName),
		"{{email}}", html.EscapeString(c.email),
		"{{address}}", html.EscapeString(c.address),
	).Replace(text)
}

type companyPageSeed struct {
	slug            string
	title           string
	seoTitle        string
	metaDescription string
	introHTML       string
	header, footer  bool
	menuOrder       int
	blocks          []model.Block
}

func companyPages(c companyDetails) []companyPageSeed {
	return []companyPageSeed{
		aboutPage(c),
		servicesPage(c),
		termsPage(c),
		privacyPage(c),
		faqPage(c),
	}
}

func aboutPage(c companyDetails) companyPageSeed {
	return companyPageSeed{
		slug:            AboutUsSlug,
		title:           "About Us",
		seoTitle:        "About " + c.name + " | Crew manning agency",
		metaDescription: "Learn who we are, what we stand for and how " + c.name + " connects qualified, STCW-compliant seafarers with shipping and offshore operators.",
		header:          true,
		menuOrder:       60,
		blocks: []model.Block{
			richText("story", c.fill(`<h2>Our story</h2>
<p>{{name}} is a crew manning agency that connects skilled seafarers with shipping and offshore operators. We recruit, screen and prepare deck, engine and catering crew, and we look after their documents and certificates so that every person who joins a vessel is qualified, compliant and ready for the job.</p>
<p>Behind every placement is a simple idea: a safe and well-run vessel starts with the right people and with clear, honest dealings between seafarers, agency and principal.</p>`)),
			richText("mission-vision", c.fill(`<h2>Mission and vision</h2>
<h3>Our mission</h3>
<p>To supply competent, certified and motivated seafarers to our principals, and to give every seafarer a fair, transparent and free-of-charge route to a career at sea.</p>
<h3>Our vision</h3>
<p>To be a crewing partner that principals and seafarers trust for professionalism, compliance and care for people.</p>`)),
			{ID: "stats", Type: model.BlockStats, Settings: &model.StatsSettings{Items: []model.StatsItemSetting{
				{Number: 16, Label: "Ranks we supply across deck, engine and catering"},
				{Number: 3, Label: "Departments covered: deck, engine and catering"},
				{Number: 24, Suffix: "/7", Label: "Operational support for crew on board"},
				{Number: 100, Suffix: "%", Label: "Mandatory documents checked before deployment"},
			}}},
			richText("values-heading", `<h2>Our values</h2>`),
			{ID: "values", Type: model.BlockCardGrid, Settings: &model.CardGridSettings{Columns: 2, Cards: []model.Card{
				{Icon: "shield", Title: "Safety first", Text: "Safe operations begin with qualified crew. We verify certificates and medical fitness before any deployment."},
				{Icon: "check", Title: "Integrity", Text: "We are honest with seafarers and principals, and we never charge applicants a fee."},
				{Icon: "award", Title: "Professionalism", Text: "We hold ourselves to the standards of the industry and to the conventions that protect seafarers."},
				{Icon: "users", Title: "Care for people", Text: "Seafarers and their families are at the centre of what we do, from the first application to sign-off."},
			}}},
		},
	}
}

func servicesPage(c companyDetails) companyPageSeed {
	return companyPageSeed{
		slug:            ServicesSlug,
		title:           "Services",
		seoTitle:        "Crew manning and crew management services | " + c.name,
		metaDescription: "Crew manning, crew management, recruitment and selection, document and certification handling, and travel and logistics for shipping and offshore operators.",
		introHTML:       c.fill(`<p>{{name}} supports principals and seafarers from recruitment to sign-off.</p>`),
		header:          true,
		menuOrder:       70,
		blocks: []model.Block{
			{ID: "services", Type: model.BlockCardGrid, Settings: &model.CardGridSettings{Columns: 3, Cards: []model.Card{
				{Icon: "ship", Title: "Crew manning", Text: "Qualified officers and ratings for offshore support vessels, tankers, bulk carriers and other ships, matched to the rank, vessel type and requirements of the principal.", Link: "/contact"},
				{Icon: "users", Title: "Crew management", Text: "Administration of contracts, wages and allowances, status tracking and welfare support for crew while they are on board and between contracts."},
				{Icon: "briefcase", Title: "Recruitment and selection", Text: "Open vacancies, structured screening and interviews, and a clear pipeline for every applicant. Recruitment is always free for seafarers.", Link: "/career"},
				{Icon: "file-text", Title: "Document and certification handling", Text: "Checking and tracking passports, seaman books, medical certificates and STCW certificates, with early warnings before anything expires."},
				{Icon: "map-pin", Title: "Travel and logistics", Text: "Joining and repatriation travel, visas and port arrangements for crew changes, planned together with the principal."},
				{Icon: "clock", Title: "Rotation and relief planning", Text: "Planned reliefs and crew changes so that a qualified replacement is lined up before a contract ends."},
			}}},
			{ID: "cta", Type: model.BlockCTA, Settings: &model.CTASettings{
				Heading:         "Ready to join our crew?",
				Text:            "Browse our current vacancies and apply online. We never charge applicants a fee.",
				ButtonLabel:     "View vacancies",
				ButtonLink:      "/career",
				BackgroundColor: "#0B2545",
			}},
		},
	}
}

func termsPage(c companyDetails) companyPageSeed {
	return companyPageSeed{
		slug:            TermsSlug,
		title:           "Terms and Conditions",
		seoTitle:        "Terms and Conditions | " + c.name,
		metaDescription: "The terms for using the " + c.name + " website and applying for vacancies: accurate information, no fees for applicants, intellectual property, liability and governing law.",
		footer:          true,
		menuOrder:       80,
		blocks:          []model.Block{richText("terms", c.fill(termsHTML))},
	}
}

func privacyPage(c companyDetails) companyPageSeed {
	return companyPageSeed{
		slug:            PrivacySlug,
		title:           "Privacy Policy",
		seoTitle:        "Privacy Policy | " + c.name,
		metaDescription: "How " + c.name + " collects, uses, shares, stores and protects personal data from our contact and application forms, and your rights under Indonesia's Personal Data Protection Law (UU PDP).",
		footer:          true,
		menuOrder:       90,
		blocks:          []model.Block{richText("privacy", c.fill(privacyHTML))},
	}
}

func faqPage(c companyDetails) companyPageSeed {
	items := make([]model.FAQItem, len(faqItems))
	for i, f := range faqItems {
		items[i] = model.FAQItem{Question: f.question, AnswerHTML: c.fill(f.answer)}
	}
	return companyPageSeed{
		slug:            FAQSlug,
		title:           "FAQ",
		seoTitle:        "Frequently asked questions | " + c.name,
		metaDescription: "Answers about applying for a seafarer job with " + c.name + ": the application process, required documents, fees, contract length and your data.",
		introHTML:       c.fill(`<p>Answers to the questions seafarers ask us most often.</p>`),
		menuOrder:       100,
		blocks:          []model.Block{{ID: "faq", Type: model.BlockFAQ, Settings: &model.FAQSettings{Items: items}}},
	}
}

func richText(id, htmlText string) model.Block {
	return model.Block{ID: id, Type: model.BlockRichText, Settings: &model.RichTextSettings{HTML: htmlText}}
}

// Slugs of the seeded company pages.
const (
	AboutUsSlug  = "about-us"
	ServicesSlug = "services"
	TermsSlug    = "terms-and-conditions"
	PrivacySlug  = "privacy-policy"
	FAQSlug      = "faq"
)

func (p companyPageSeed) request() (request.Page, error) {
	raws := make([]json.RawMessage, len(p.blocks))
	for i, b := range p.blocks {
		raw, err := json.Marshal(b)
		if err != nil {
			return request.Page{}, fmt.Errorf("encode block %q: %w", b.ID, err)
		}
		raws[i] = raw
	}
	return request.Page{
		Title:           p.title,
		Slug:            p.slug,
		SEOTitle:        p.seoTitle,
		MetaDescription: p.metaDescription,
		Status:          model.StatusPublished,
		InHeaderMenu:    p.header,
		InFooterMenu:    p.footer,
		MenuOrder:       p.menuOrder,
		IntroHTML:       p.introHTML,
		Blocks:          raws,
	}, nil
}

func (p companyPageSeed) toPage() (model.Page, error) {
	req, err := p.request()
	if err != nil {
		return model.Page{}, err
	}
	if problems := req.Validate(); len(problems) > 0 {
		return model.Page{}, fmt.Errorf("page %q is invalid: %v", p.slug, problems)
	}
	return req.Model(), nil
}

func newPageService(pool *pgxpool.Pool, site service.SiteChanger, now func() time.Time) *service.PageService {
	lookup := repository.NewCMSLookupRepository(pool)
	return service.NewPageService(repository.NewPageRepository(pool), lookup, service.NewPublicFileChecker(lookup), service.NewCMSPolicy(), site, now)
}

// CompanyPages creates the company and legal pages that do not exist yet and returns how many it created.
func CompanyPages(ctx context.Context, pool *pgxpool.Pool, site service.SiteChanger, now func() time.Time) (int, error) {
	pages := newPageService(pool, site, now)
	created := 0
	for _, seed := range companyPages(defaultCompanyDetails()) {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pages WHERE slug = $1)`, seed.slug).Scan(&exists); err != nil {
			return created, fmt.Errorf("check page %q: %w", seed.slug, err)
		}
		if exists {
			continue
		}
		page, err := seed.toPage()
		if err != nil {
			return created, err
		}
		if _, err := pages.Create(ctx, page); err != nil {
			return created, fmt.Errorf("create page %q: %w", seed.slug, err)
		}
		created++
	}
	return created, nil
}
