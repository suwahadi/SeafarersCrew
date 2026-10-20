package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type memTemplates struct {
	rows map[uuid.UUID]model.DocumentTemplate
}

func newMemTemplates() *memTemplates {
	return &memTemplates{rows: map[uuid.UUID]model.DocumentTemplate{}}
}

func (m *memTemplates) List(_ context.Context, kind string) ([]model.DocumentTemplate, error) {
	var out []model.DocumentTemplate
	for _, t := range m.rows {
		if kind == "" || t.Kind == kind {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memTemplates) Get(_ context.Context, id uuid.UUID) (model.DocumentTemplate, error) {
	t, ok := m.rows[id]
	if !ok {
		return t, repository.ErrNotFound
	}
	return t, nil
}

func (m *memTemplates) Default(_ context.Context, kind string) (model.DocumentTemplate, error) {
	for _, t := range m.rows {
		if t.Kind == kind && t.IsDefault {
			return t, nil
		}
	}
	return model.DocumentTemplate{}, repository.ErrNotFound
}

func (m *memTemplates) CountByKind(_ context.Context, kind string) (int, error) {
	n := 0
	for _, t := range m.rows {
		if t.Kind == kind {
			n++
		}
	}
	return n, nil
}

func (m *memTemplates) Save(_ context.Context, t model.DocumentTemplate) (model.DocumentTemplate, error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.IsDefault {
		for id, o := range m.rows {
			if o.Kind == t.Kind && id != t.ID {
				o.IsDefault = false
				m.rows[id] = o
			}
		}
	}
	m.rows[t.ID] = t
	return t, nil
}

func (m *memTemplates) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.rows, id)
	return nil
}

func tpl(name, kind string, isDefault bool) model.DocumentTemplate {
	return model.DocumentTemplate{Name: name, Kind: kind, PaperSize: "a4", Orientation: "portrait", BodyHTML: "<p>{crew_name}</p>", IsDefault: isDefault}
}

func TestFirstTemplateOfAKindBecomesDefault(t *testing.T) {
	svc := NewDocumentTemplateService(newMemTemplates())
	res, err := svc.Save(context.Background(), tpl("SEA", model.DocumentSEA, false))
	require.NoError(t, err)
	assert.True(t, res.Template.IsDefault)
}

func TestSettingADefaultUnsetsThePreviousOne(t *testing.T) {
	store := newMemTemplates()
	svc := NewDocumentTemplateService(store)
	ctx := context.Background()
	first, err := svc.Save(ctx, tpl("A", model.DocumentCV, true))
	require.NoError(t, err)
	second, err := svc.Save(ctx, tpl("B", model.DocumentCV, true))
	require.NoError(t, err)
	assert.False(t, store.rows[first.Template.ID].IsDefault)
	assert.True(t, store.rows[second.Template.ID].IsDefault)

	resolved, err := svc.Resolve(ctx, model.DocumentCV, nil)
	require.NoError(t, err)
	assert.Equal(t, second.Template.ID, resolved.ID)

	_, err = svc.Resolve(ctx, model.DocumentSEA, &first.Template.ID)
	requireAppErr(t, err, "validation_failed")
}

func TestDefaultTemplateCannotBeUnsetOrDeleted(t *testing.T) {
	svc := NewDocumentTemplateService(newMemTemplates())
	ctx := context.Background()
	def, err := svc.Save(ctx, tpl("A", model.DocumentSEA, true))
	require.NoError(t, err)

	update := def.Template
	update.IsDefault = false
	_, err = svc.Save(ctx, update)
	requireAppErr(t, err, "validation_failed")
	requireAppErr(t, svc.Delete(ctx, def.Template.ID), "default_template")

	other, err := svc.Save(ctx, tpl("B", model.DocumentSEA, false))
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, other.Template.ID))
}

func TestLastTemplateOfAKindCannotBeDeleted(t *testing.T) {
	store := newMemTemplates()
	svc := NewDocumentTemplateService(store)
	ctx := context.Background()
	only := tpl("Only", model.DocumentQuotation, false)
	only.ID = uuid.New()
	store.rows[only.ID] = only
	requireAppErr(t, svc.Delete(ctx, only.ID), "last_template")
	requireAppErr(t, svc.Delete(ctx, uuid.New()), "not_found")
}

func TestSaveSanitizesHTMLKeepsKindAndWarnsAboutUnknownTags(t *testing.T) {
	svc := NewDocumentTemplateService(newMemTemplates())
	ctx := context.Background()
	in := tpl("CV", model.DocumentCV, false)
	in.BodyHTML = `<p onclick="x()">{crew_name} {grand_total}</p><script>bad()</script>`
	res, err := svc.Save(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, "<p>{crew_name} {grand_total}</p>", res.Template.BodyHTML)
	assert.Equal(t, []string{"grand_total"}, res.UnknownTags)

	change := res.Template
	change.Kind = model.DocumentSEA
	saved, err := svc.Save(ctx, change)
	require.NoError(t, err)
	assert.Equal(t, model.DocumentCV, saved.Template.Kind, "the kind of a template is fixed")

	empty := tpl("Empty", model.DocumentCV, false)
	empty.BodyHTML = "<script>x</script>"
	_, err = svc.Save(ctx, empty)
	requireAppErr(t, err, "validation_failed")
}
