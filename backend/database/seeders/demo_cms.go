package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type heroSeed struct {
	heading, subheading, label, link string
}

var demoHeroSlides = []heroSeed{
	{"Qualified seafarers for every voyage", "Deck, engine and catering crew, screened and certified to STCW standards.", "View vacancies", "/career"},
	{"Your career at sea starts here", "Apply online in minutes. Recruitment is always free for seafarers.", "Apply now", "/career"},
	{"Crew management you can rely on", "Contracts, documents and reliefs planned ahead, from sign-on to sign-off.", "Our services", "/services"},
}

type gallerySeed struct {
	caption, alt string
	onHome       bool
}

var demoGallery = []gallerySeed{
	{"Open sea swell", "Layered ocean waves under a pale sky", true},
	{"Anchor symbol", "Orange anchor symbol against a dark sea", false},
	{"Sunrise at sea", "Sun rising over the sea horizon", true},
	{"Cargo ship underway", "Ship silhouette sailing along the horizon", true},
	{"Navigation compass", "Compass rose on a deep blue background", false},
	{"Evening swell", "Dark ocean waves under a warm evening sky", true},
	{"Anchored for the night", "Anchor symbol above a dark wave line", false},
	{"Dawn on the Java Sea", "Sun low over calm water at dawn", true},
	{"Vessel on passage", "Vessel on passage under a clear sky", true},
	{"Course plotted", "Compass rose with north and east points highlighted", false},
	{"Rolling swell", "Rolling ocean swell in layered blue tones", false},
	{"Safe at anchor", "Anchor symbol against a twilight sky", true},
}

type testimonialSeed struct {
	name, position, quote string
	rating                int
}

var demoTestimonials = []testimonialSeed{
	{"Capt. Hendra Wijaya", "Master, AHTS vessel", "The team prepared my documents early and my joining went smoothly. Communication was clear from the first call to sign-on.", 5},
	{"Rizky Pratama", "Chief Engineer", "A professional agency that respects seafarers. No fees, no surprises, and my contract matched what was promised.", 5},
	{"Maria Santos", "Able Seaman", "I always knew the status of my application and my relief was planned well before my contract ended.", 4},
	{"Andi Susanto", "Crewing Manager, offshore operator", "Reliable crew who arrive with complete and valid certificates. It saves our operations team a lot of follow-up.", 5},
	{"Oleksandr Kovalenko", "Master, tanker", "Good planning of crew changes and a responsive team when schedules shift at short notice.", 4},
	{"Dewi Lestari", "Cook", "Friendly and fair. They explained the contract and allowances in detail before I signed.", 5},
}

var demoClientNames = []string{
	"Samudera Nusantara Offshore", "Bahari Lintas Energi", "Meridian Marine Services", "Karya Tirta Logistik",
	"Fjord Shipping Management", "Pelita Samudra Tankers", "Nusa Gas Carriers", "Arunika Dredging",
}

type statsSeed struct {
	number        int
	suffix, label string
}

var demoStats = []statsSeed{
	{650, "+", "Seafarers Deployed"},
	{40, "+", "Principals Served"},
	{120, "+", "Vessels Crewed"},
	{12, "+", "Years of Experience"},
}

func seedDemoCMS(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	images, err := d.loadImages(ctx)
	if err != nil {
		return err
	}
	for i, h := range demoHeroSlides {
		if _, err := d.CMS.HeroSlides.Create(ctx, model.HeroSlide{
			MasterBase: activeAt(i), ImageFileID: images.hero[i], Heading: h.heading, Subheading: h.subheading, ButtonLabel: h.label, ButtonLink: h.link,
		}); err != nil {
			return fmt.Errorf("create hero slide %d: %w", i+1, err)
		}
	}
	gallery := make([]model.GalleryImage, len(demoGallery))
	for i, g := range demoGallery {
		gallery[i] = model.GalleryImage{MasterBase: activeAt(i), ImageFileID: images.gallery[i], Caption: g.caption, AltText: g.alt, ShowOnHome: g.onHome}
	}
	if _, err := d.CMS.Gallery.BulkCreate(ctx, gallery); err != nil {
		return fmt.Errorf("create gallery images: %w", err)
	}
	for i, t := range demoTestimonials {
		if _, err := d.CMS.Testimonials.Create(ctx, model.Testimonial{
			MasterBase: activeAt(i), Name: t.name, Position: t.position, Quote: t.quote, Rating: t.rating,
		}); err != nil {
			return fmt.Errorf("create testimonial %d: %w", i+1, err)
		}
	}
	for i, name := range demoClientNames {
		if _, err := d.CMS.ClientLogos.Create(ctx, model.ClientLogo{
			MasterBase: activeAt(i), Name: name, LogoFileID: images.logos[i], URL: clientURL(name),
		}); err != nil {
			return fmt.Errorf("create client logo %d: %w", i+1, err)
		}
	}
	for i, s := range demoStats {
		if _, err := d.CMS.Stats.Create(ctx, model.StatsItem{MasterBase: activeAt(i), Number: s.number, Suffix: s.suffix, Label: s.label}); err != nil {
			return fmt.Errorf("create stats item %d: %w", i+1, err)
		}
	}
	return nil
}

func activeAt(index int) model.MasterBase {
	return model.MasterBase{IsActive: true, SortOrder: (index + 1) * 10}
}

func clientURL(name string) string {
	slug := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			slug = append(slug, r)
		case r >= 'A' && r <= 'Z':
			slug = append(slug, r+('a'-'A'))
		case r == ' ' && len(slug) > 0 && slug[len(slug)-1] != '-':
			slug = append(slug, '-')
		}
	}
	return "https://" + string(slug) + ".example.com"
}
