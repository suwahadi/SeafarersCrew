package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// CV import decision actions.
const (
	CVActionMatch  = "match"
	CVActionCreate = "create"
	CVActionSkip   = "skip"
)

const (
	cvChoiceCertificate = "certificate"
	cvChoiceDocument    = "document"
	cvChoiceRank        = "rank"
	cvChoiceVesselType  = "vessel_type"
	cvTableDocuments    = "documents"
	cvTableCerts        = "certificates"
	cvDateLayout        = "2006-01-02"
)

// CrewCVStore is the data access a CV import needs.
type CrewCVStore interface {
	Lookups(ctx context.Context) (repository.CVLookups, error)
	Duplicates(ctx context.Context, k repository.CVDuplicateKeys) ([]repository.CVDuplicate, error)
	Create(ctx context.Context, in repository.CVCreate, newID func() (string, error), attempts int) (string, error)
}

// CVFileStore stores and removes the uploaded CV file.
type CVFileStore interface {
	Upload(ctx context.Context, in UploadInput) (model.File, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// OptionsInvalidator drops cached master options after master data changes.
type OptionsInvalidator interface {
	Invalidate(ctx context.Context) error
}

// CrewCVImportService turns one RVOS CV into a crew profile after a preview.
type CrewCVImportService struct {
	store   CrewCVStore
	cache   Cache
	files   CVFileStore
	options OptionsInvalidator
	cpu     Bounded
	newID   func() (string, error)
}

// NewCrewCVImportService builds a CrewCVImportService.
func NewCrewCVImportService(store CrewCVStore, cache Cache, files CVFileStore, options OptionsInvalidator, cpu Bounded) *CrewCVImportService {
	return &CrewCVImportService{store: store, cache: cache, files: files, options: options, cpu: cpu, newID: publicid.New}
}

// CVChoice is a CV text the admin must map before confirming.
type CVChoice struct {
	Key         string     `json:"key"`
	Kind        string     `json:"kind"`
	Text        string     `json:"text"`
	SuggestedID *uuid.UUID `json:"suggested_id,omitempty"`
	AllowCreate bool       `json:"allow_create"`
}

// CVField is one personal value shown in the preview.
type CVField struct {
	Label     string `json:"label"`
	Value     string `json:"value"`
	ChoiceKey string `json:"choice_key,omitempty"`
}

// CVRow is one previewed table row.
type CVRow struct {
	Row        int               `json:"row"`
	Cells      map[string]string `json:"cells"`
	ChoiceKeys []string          `json:"choice_keys,omitempty"`
	MatchedAs  string            `json:"matched_as,omitempty"`
	Warnings   []string          `json:"warnings,omitempty"`
}

// CVPreview is what a validated CV would create.
type CVPreview struct {
	Token        string                  `json:"token"`
	FullName     string                  `json:"full_name"`
	Personal     []CVField               `json:"personal"`
	NextOfKin    []CVRow                 `json:"next_of_kin"`
	Documents    []CVRow                 `json:"documents"`
	Certificates []CVRow                 `json:"certificates"`
	SeaService   []CVRow                 `json:"sea_service"`
	NotImported  []string                `json:"not_imported"`
	Warnings     []string                `json:"warnings"`
	Errors       []CVCellError           `json:"errors"`
	Duplicate    *repository.CVDuplicate `json:"duplicate,omitempty"`
	Choices      []CVChoice              `json:"choices"`
}

// CVDecision is the admin's answer to one choice.
type CVDecision struct {
	Action   string `json:"action"`
	TargetID string `json:"target_id"`
}

// CVCommitInput confirms a staged CV import.
type CVCommitInput struct {
	Token     string                `json:"token"`
	Decisions map[string]CVDecision `json:"decisions"`
}

// CVCommitResult names the crew member created from a CV.
type CVCommitResult struct {
	CrewID   string `json:"crew_id"`
	FullName string `json:"full_name"`
}

type cvPlanRecord struct {
	Table      string             `json:"table"`
	Text       string             `json:"text"`
	Competency bool               `json:"competency"`
	Match      *cvMatch           `json:"match,omitempty"`
	ChoiceKey  string             `json:"choice_key,omitempty"`
	Doc        model.CrewDocument `json:"doc"`
}

type cvPlanSea struct {
	Experience model.Experience `json:"experience"`
	RankKey    string           `json:"rank_key,omitempty"`
	VesselKey  string           `json:"vessel_key,omitempty"`
	VesselText string           `json:"vessel_text,omitempty"`
	Flag       string           `json:"flag,omitempty"`
}

type cvPlan struct {
	Crew    model.Crew                 `json:"crew"`
	RankKey string                     `json:"rank_key,omitempty"`
	Keys    repository.CVDuplicateKeys `json:"keys"`
	Kins    []model.NextOfKin          `json:"kins"`
	Records []cvPlanRecord             `json:"records"`
	Sea     []cvPlanSea                `json:"sea"`
	Choices []CVChoice                 `json:"choices"`
}

type stagedCV struct {
	Plan     cvPlan `json:"plan"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

type cvResolution struct {
	Action string
	Kind   string
	ID     uuid.UUID
}

func cvImportKey(tok string) string { return "cv_import:" + tok }

func cvExpired() error {
	return apperr.New(http.StatusGone, "import_expired", "The preview expired. Upload the CV again.")
}

func cvDecisionsError() error {
	return apperr.Validation(map[string]string{"decisions": "Choose what to do with every highlighted item."})
}

func cvBefore(a, b *time.Time) bool { return a != nil && b != nil && a.Before(*b) }

func cvClamp(s string, n int) string {
	if utf8.RuneCountInString(s) > n {
		return string([]rune(s)[:n])
	}
	return s
}

func cvAliasKey(s string) string { return cvClamp(normalizeCVText(s), 200) }

func cvTooLong(s string, n int) bool { return utf8.RuneCountInString(s) > n }

func cvDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(cvDateLayout)
}

// Validate parses a CV, matches it against master data and stages it when nothing blocks the import.
func (s *CrewCVImportService) Validate(ctx context.Context, filename string, data []byte) (CVPreview, error) {
	ext, err := filecheck.CheckName(filename)
	if err != nil {
		return CVPreview{}, err
	}
	if err := filecheck.CheckExtension(filecheck.ContextCrewCV, ext); err != nil {
		return CVPreview{}, err
	}
	var wb CVWorkbook
	if err := s.cpu.Do(ctx, func(context.Context) error {
		if err := filecheck.CheckStoredXLSX(data); err != nil {
			return err
		}
		var err error
		wb, err = ParseCrewCV(data)
		return err
	}); err != nil {
		return CVPreview{}, err
	}
	l, err := s.store.Lookups(ctx)
	if err != nil {
		return CVPreview{}, fmt.Errorf("load cv lookups: %w", err)
	}
	b := &cvBuilder{m: newCVMatcher(l), seen: map[string]bool{}}
	plan, preview := b.build(wb, l)
	dups, err := s.store.Duplicates(ctx, plan.Keys)
	if err != nil {
		return CVPreview{}, fmt.Errorf("check cv duplicates: %w", err)
	}
	if len(dups) > 0 {
		preview.Duplicate = &dups[0]
	}
	if len(preview.Errors) > 0 || preview.Duplicate != nil {
		return preview, nil
	}
	tok, err := token.URLSafe(18)
	if err != nil {
		return CVPreview{}, fmt.Errorf("cv import token: %w", err)
	}
	payload, err := json.Marshal(stagedCV{Plan: plan, Filename: filename, Data: data})
	if err != nil {
		return CVPreview{}, fmt.Errorf("encode cv import: %w", err)
	}
	if err := s.cache.Set(ctx, cvImportKey(tok), payload, importTTL); err != nil {
		return CVPreview{}, fmt.Errorf("stage cv import: %w", err)
	}
	preview.Token = tok
	return preview, nil
}

type cvBuilder struct {
	m       cvMatcher
	seen    map[string]bool
	choices []CVChoice
}

func (b *cvBuilder) choice(kind, text string, suggested *uuid.UUID) string {
	key := kind + ":" + normalizeCVText(text)
	if !b.seen[key] {
		b.seen[key] = true
		b.choices = append(b.choices, CVChoice{Key: key, Kind: kind, Text: strings.TrimSpace(text), SuggestedID: suggested, AllowCreate: kind != cvChoiceRank})
	}
	return key
}

var cvClearable = map[string]string{
	"place_of_birth": "Place of birth", "nationality": "Nationality", "religion": "Religion", "address": "Address", "coverall_size": "PPE size",
}

func clearCrewField(in *request.Crew, field string) {
	switch field {
	case "place_of_birth":
		in.PlaceOfBirth = ""
	case "nationality":
		in.Nationality = ""
	case "religion":
		in.Religion = ""
	case "address":
		in.Address = ""
	case "coverall_size":
		in.CoverallSize = ""
	}
}

func (b *cvBuilder) build(wb CVWorkbook, l repository.CVLookups) (cvPlan, CVPreview) {
	p := wb.Personal
	preview := CVPreview{
		FullName: p.FullName, Errors: append([]CVCellError{}, wb.Errors...), NotImported: []string{}, Warnings: []string{},
		NextOfKin: []CVRow{}, Documents: []CVRow{}, Certificates: []CVRow{}, SeaService: []CVRow{},
	}
	warn := func(msg string) { preview.Warnings = append(preview.Warnings, msg) }
	for _, section := range wb.Missing {
		warn(fmt.Sprintf("No %s section was found in the CV, so nothing was imported from it.", section))
	}

	mobile, ok := normalizeCVPhone(p.MobilePhone, p.Nationality)
	if !ok {
		warn(fmt.Sprintf("Mobile number %q is not a valid international number and was left empty.", p.MobilePhone))
	}
	home, ok := normalizeCVPhone(p.HomePhone, p.Nationality)
	if !ok {
		warn(fmt.Sprintf("Home phone %q is not a valid international number and was left empty.", p.HomePhone))
	}
	marital := cvMaritalStatus(p.MaritalStatus)
	if p.MaritalStatus != "" && marital == "" {
		warn(fmt.Sprintf("Marital status %q is not recognised and was left empty.", p.MaritalStatus))
	}
	email := strings.ToLower(strings.TrimSpace(p.Email))
	if errs := (request.Errors{}); email != "" {
		errs.Email("email", email)
		if len(errs) > 0 {
			warn(fmt.Sprintf("Email %q is not valid and was left empty.", p.Email))
			email = ""
		}
	}
	in := request.Crew{
		FullName: p.FullName, PlaceOfBirth: p.PlaceOfBirth, DateOfBirth: cvDate(p.DateOfBirth), Nationality: p.Nationality,
		Religion: p.Religion, MaritalStatus: marital, CoverallSize: p.PPESize, Address: p.Address, HomePhone: home,
		MobilePhone: mobile, Email: email,
	}
	for field, msg := range in.Validate() {
		if label, ok := cvClearable[field]; ok {
			warn(fmt.Sprintf("%s was left empty: %s", label, msg))
			clearCrewField(&in, field)
		}
	}
	if strings.TrimSpace(p.FullName) == "" {
		preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionPersonal, Row: p.NameRow, Message: "The CV has no name."})
	} else if msg, bad := in.Validate()["full_name"]; bad {
		preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionPersonal, Row: p.NameRow, Message: msg})
	}
	plan := cvPlan{Crew: in.Model(), Keys: repository.CVDuplicateKeys{
		FullName: strings.TrimSpace(p.FullName), DateOfBirth: p.DateOfBirth, PassportType: l.PassportType, SeamanBookType: l.SeamanBookType,
	}}
	rankField := CVField{Label: "Rank", Value: p.Rank}
	if p.Rank != "" {
		if mt, ok := b.m.rank(p.Rank); ok {
			id := mt.ID
			plan.Crew.AppliedRankID = &id
		} else {
			plan.RankKey = b.choice(cvChoiceRank, p.Rank, nil)
			rankField.ChoiceKey = plan.RankKey
		}
	}
	preview.Personal = []CVField{
		{Label: "Name", Value: p.FullName}, rankField, {Label: "Date of birth", Value: cvDate(p.DateOfBirth)},
		{Label: "Place of birth", Value: in.PlaceOfBirth}, {Label: "Marital status", Value: marital},
		{Label: "Nationality", Value: in.Nationality}, {Label: "Religion", Value: in.Religion}, {Label: "Email", Value: email},
		{Label: "PPE size", Value: in.CoverallSize}, {Label: "Home phone", Value: home}, {Label: "Mobile phone", Value: mobile},
		{Label: "Address", Value: in.Address},
	}
	if p.BloodGroup != "" {
		preview.NotImported = append(preview.NotImported, "Blood group")
	}
	if p.DateOfJoin != "" {
		preview.NotImported = append(preview.NotImported, "Date of join")
	}
	b.kins(wb, &plan, &preview, warn)
	b.documents(wb, l, &plan, &preview)
	b.certificates(wb, &plan, &preview)
	b.seaService(wb, &plan, &preview)
	plan.Choices = b.choices
	preview.Choices = b.choices
	if preview.Choices == nil {
		preview.Choices = []CVChoice{}
	}
	return plan, preview
}

func (b *cvBuilder) kins(wb CVWorkbook, plan *cvPlan, preview *CVPreview, warn func(string)) {
	add := func(k model.NextOfKin, original string) {
		if cvTooLong(k.Name, 150) {
			warn("A next of kin name is longer than 150 characters and was not imported.")
			return
		}
		if cvTooLong(k.Address, 500) {
			warn("A next of kin address is longer than 500 characters and was left empty.")
			k.Address = ""
		}
		plan.Kins = append(plan.Kins, k)
		preview.NextOfKin = append(preview.NextOfKin, CVRow{Cells: map[string]string{
			"name": k.Name, "relationship": k.Relationship, "relationship_text": original, "phone": k.Phone, "address": k.Address,
		}})
	}
	kin := wb.NextOfKin
	if kin.Name != "" {
		phone, ok := normalizeCVPhone(kin.Phone, wb.Personal.Nationality)
		if !ok {
			warn(fmt.Sprintf("Next of kin phone %q is not a valid international number and was left empty.", kin.Phone))
		}
		add(model.NextOfKin{Name: kin.Name, Relationship: mapCVRelationship(kin.Relationship), Phone: phone, Address: kin.Address, IsPrimary: true}, kin.Relationship)
	}
	for _, parent := range []string{wb.Father, wb.Mother} {
		if parent == "" || normalizeCVText(parent) == normalizeCVText(kin.Name) {
			continue
		}
		add(model.NextOfKin{Name: parent, Relationship: "parent"}, "")
	}
}

func (b *cvBuilder) documents(wb CVWorkbook, l repository.CVLookups, plan *cvPlan, preview *CVPreview) {
	for _, d := range wb.Documents {
		label := strings.TrimSpace(d.Label)
		fail := func(msg string) {
			preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionDocuments, Row: d.Row, Message: msg})
		}
		if label == "" {
			fail("The document type is missing.")
		}
		if cvTooLong(d.Number, 60) {
			fail("The number may not be longer than 60 characters.")
		}
		if cvBefore(d.Expires, d.Issued) {
			fail("The expiry date must be on or after the issued date.")
		}
		place := d.PlaceOfIssue
		if cvTooLong(place, 100) {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Documents row %d: the place of issue is longer than 100 characters and was left empty.", d.Row))
			place = ""
		}
		rec := cvPlanRecord{Table: cvTableDocuments, Text: label, Doc: model.CrewDocument{
			Number: d.Number, IssuedDate: d.Issued, ExpiredDate: d.Expires, PlaceOfIssue: place,
		}}
		row := CVRow{Row: d.Row, Cells: map[string]string{
			"type": label, "number": d.Number, "issued": cvDate(d.Issued), "expires": cvDate(d.Expires), "place": place,
		}}
		if mt, ok := b.m.document(label); ok {
			rec.Match = &mt
			row.MatchedAs = b.m.name(mt)
			switch mt.ID {
			case l.PassportType:
				plan.Keys.PassportNumber = d.Number
			case l.SeamanBookType:
				plan.Keys.SeamanBookNumber = d.Number
			}
		} else if label != "" {
			rec.ChoiceKey = b.choice(cvChoiceDocument, label, nil)
			row.ChoiceKeys = []string{rec.ChoiceKey}
		}
		plan.Records = append(plan.Records, rec)
		preview.Documents = append(preview.Documents, row)
	}
}

func (b *cvBuilder) certificates(wb CVWorkbook, plan *cvPlan, preview *CVPreview) {
	for _, c := range wb.Certificates {
		if cvBefore(c.Expires, c.Issued) {
			preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionCerts, Row: c.Row, Message: "The expiry date must be on or after the issued date."})
		}
		rec := cvPlanRecord{Table: cvTableCerts, Text: c.Text, Competency: c.Competency, Doc: model.CrewDocument{
			Number: "-", IssuedDate: c.Issued, ExpiredDate: c.Expires,
		}}
		row := CVRow{Row: c.Row, Cells: map[string]string{"text": c.Text, "issued": cvDate(c.Issued), "expires": cvDate(c.Expires)}}
		if mt, ok := b.m.certificate(c.Text); ok {
			rec.Match = &mt
			row.MatchedAs = b.m.name(mt)
		} else {
			rec.ChoiceKey = b.choice(cvChoiceCertificate, c.Text, nil)
			row.ChoiceKeys = []string{rec.ChoiceKey}
		}
		plan.Records = append(plan.Records, rec)
		preview.Certificates = append(preview.Certificates, row)
	}
}

func (b *cvBuilder) seaService(wb CVWorkbook, plan *cvPlan, preview *CVPreview) {
	for _, x := range wb.SeaService {
		if cvTooLong(x.Vessel, 150) {
			preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionSeaService, Row: x.Row, Message: "The vessel name may not be longer than 150 characters."})
		}
		if cvBefore(x.To, x.From) {
			preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionSeaService, Row: x.Row, Message: "The sign-off date must be on or after the sign-on date."})
		}
		if cvTooLong(x.Company, 150) {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Sea service row %d: the company is longer than 150 characters and was left empty.", x.Row))
			x.Company = ""
		}
		if cvTooLong(x.GRTHP, 60) {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Sea service row %d: the GRT / HP value is longer than 60 characters and was left empty.", x.Row))
			x.GRTHP = ""
		}
		row := CVRow{Row: x.Row, Cells: map[string]string{
			"vessel": x.Vessel, "rank": x.Rank, "flag": x.Flag, "vessel_type": x.VesselType, "grt_hp": x.GRTHP,
			"company": x.Company, "from": cvDate(x.From), "to": cvDate(x.To),
		}}
		sp := cvPlanSea{Flag: x.Flag, VesselText: x.VesselType, Experience: model.Experience{
			VesselName: x.Vessel, GRTHP: x.GRTHP, SignOffDate: x.To, Status: "on_board",
		}}
		if x.To != nil {
			sp.Experience.Status = "sign_off"
		}
		if x.From == nil {
			preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionSeaService, Row: x.Row, Message: "The sign-on date is missing."})
		} else {
			sp.Experience.SignOnDate = *x.From
		}
		if id := b.m.company(x.Company); id != nil {
			sp.Experience.CompanyID = id
		} else {
			sp.Experience.CompanyName = x.Company
		}
		if x.Rank == "" {
			preview.Errors = append(preview.Errors, CVCellError{Section: cvSectionSeaService, Row: x.Row, Message: "The rank is missing."})
		} else if mt, ok := b.m.rank(x.Rank); ok {
			sp.Experience.RankID = mt.ID
		} else {
			sp.RankKey = b.choice(cvChoiceRank, x.Rank, nil)
			row.ChoiceKeys = append(row.ChoiceKeys, sp.RankKey)
		}
		if x.VesselType != "" {
			mt, ok := b.m.vesselType(x.VesselType)
			switch {
			case ok && mt.By != "prefix":
				id := mt.ID
				sp.Experience.VesselTypeID = &id
			case ok:
				id := mt.ID
				sp.VesselKey = b.choice(cvChoiceVesselType, x.VesselType, &id)
			default:
				sp.VesselKey = b.choice(cvChoiceVesselType, x.VesselType, nil)
			}
			if sp.VesselKey != "" {
				row.ChoiceKeys = append(row.ChoiceKeys, sp.VesselKey)
			}
		}
		plan.Sea = append(plan.Sea, sp)
		preview.SeaService = append(preview.SeaService, row)
	}
}

// Commit writes a staged CV import using the admin's decisions.
func (s *CrewCVImportService) Commit(ctx context.Context, in CVCommitInput) (CVCommitResult, error) {
	if in.Token == "" {
		return CVCommitResult{}, cvExpired()
	}
	raw, ok, err := s.cache.Get(ctx, cvImportKey(in.Token))
	if err != nil {
		return CVCommitResult{}, fmt.Errorf("load staged cv import: %w", err)
	}
	if !ok {
		return CVCommitResult{}, cvExpired()
	}
	var staged stagedCV
	if err := json.Unmarshal(raw, &staged); err != nil {
		return CVCommitResult{}, cvExpired()
	}
	l, err := s.store.Lookups(ctx)
	if err != nil {
		return CVCommitResult{}, fmt.Errorf("load cv lookups: %w", err)
	}
	res, err := resolveCVDecisions(staged.Plan.Choices, in.Decisions, l)
	if err != nil {
		return CVCommitResult{}, err
	}
	if err := s.cache.Delete(ctx, cvImportKey(in.Token)); err != nil {
		return CVCommitResult{}, fmt.Errorf("consume cv import: %w", err)
	}
	file, err := s.files.Upload(ctx, UploadInput{Context: filecheck.ContextCrewCV, Filename: staged.Filename, Data: staged.Data, OwnerType: "crews"})
	if err != nil {
		return CVCommitResult{}, err
	}
	create := buildCVCreate(staged.Plan, res, file.ID)
	pid, err := s.store.Create(ctx, create, s.newID, publicid.MaxAttempts)
	if err != nil {
		if delErr := s.files.Delete(ctx, file.ID); delErr != nil {
			logctx.From(ctx).WarnContext(ctx, "cv file not removed after a failed import", "error", delErr)
		}
		if errors.Is(err, repository.ErrCVDuplicate) {
			return CVCommitResult{}, apperr.Conflict("crew_exists", "This crew member is already registered.")
		}
		return CVCommitResult{}, fmt.Errorf("create crew from cv: %w", err)
	}
	if len(create.NewTypes) > 0 {
		if err := s.options.Invalidate(ctx); err != nil {
			logctx.From(ctx).WarnContext(ctx, "master options not refreshed after a cv import", "error", err)
		}
	}
	return CVCommitResult{CrewID: pid, FullName: create.Crew.FullName}, nil
}

func cvContains(items []repository.CVMasterItem, id uuid.UUID) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

func cvTargetKind(choiceKind string, id uuid.UUID, l repository.CVLookups) (string, bool) {
	switch choiceKind {
	case cvChoiceCertificate:
		if cvContains(l.CertificateTypes, id) {
			return repository.CVKindCertificateType, true
		}
		if cvContains(l.DocumentTypes, id) {
			return repository.CVKindDocumentType, true
		}
	case cvChoiceDocument:
		if cvContains(l.DocumentTypes, id) {
			return repository.CVKindDocumentType, true
		}
	case cvChoiceRank:
		if cvContains(l.Ranks, id) {
			return repository.CVKindRank, true
		}
	case cvChoiceVesselType:
		if cvContains(l.VesselTypes, id) {
			return repository.CVKindVesselType, true
		}
	}
	return "", false
}

var cvNewKinds = map[string]string{
	cvChoiceCertificate: repository.CVKindCertificateType, cvChoiceDocument: repository.CVKindDocumentType, cvChoiceVesselType: repository.CVKindVesselType,
}

func resolveCVDecisions(choices []CVChoice, decisions map[string]CVDecision, l repository.CVLookups) (map[string]cvResolution, error) {
	out := make(map[string]cvResolution, len(choices))
	for _, c := range choices {
		d, ok := decisions[c.Key]
		if !ok {
			return nil, cvDecisionsError()
		}
		switch d.Action {
		case CVActionSkip:
			out[c.Key] = cvResolution{Action: CVActionSkip}
		case CVActionCreate:
			if !c.AllowCreate {
				return nil, cvDecisionsError()
			}
			out[c.Key] = cvResolution{Action: CVActionCreate, Kind: cvNewKinds[c.Kind]}
		case CVActionMatch:
			id, err := uuid.Parse(d.TargetID)
			if err != nil {
				return nil, cvDecisionsError()
			}
			kind, ok := cvTargetKind(c.Kind, id, l)
			if !ok {
				return nil, cvDecisionsError()
			}
			out[c.Key] = cvResolution{Action: CVActionMatch, Kind: kind, ID: id}
		default:
			return nil, cvDecisionsError()
		}
	}
	return out, nil
}

func cvSlug(s string) string {
	var b strings.Builder
	underscore := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			underscore = false
		} else if !underscore && b.Len() > 0 {
			b.WriteByte('_')
			underscore = true
		}
	}
	slug := b.String()
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return strings.Trim(slug, "_")
}

func cvTypeName(s string) string { return cvClamp(strings.TrimSpace(s), 160) }

func buildCVCreate(plan cvPlan, res map[string]cvResolution, fileID uuid.UUID) repository.CVCreate {
	crew := plan.Crew
	crew.CVFileID = &fileID
	if plan.RankKey != "" {
		if r := res[plan.RankKey]; r.Action == CVActionMatch {
			id := r.ID
			crew.AppliedRankID = &id
		}
	}
	create := repository.CVCreate{Crew: crew, Keys: plan.Keys, Kins: plan.Kins}
	for _, c := range plan.Choices {
		r := res[c.Key]
		switch r.Action {
		case CVActionCreate:
			nt := repository.CVNewType{Key: c.Key, Kind: r.Kind, Name: cvTypeName(c.Text)}
			if r.Kind != repository.CVKindVesselType {
				nt.Code = cvSlug(c.Text)
			}
			if r.Kind == repository.CVKindCertificateType {
				nt.Category = "other"
			}
			for _, rec := range plan.Records {
				if rec.ChoiceKey != c.Key {
					continue
				}
				if rec.Competency && r.Kind == repository.CVKindCertificateType {
					nt.Category = "coc"
				}
				if rec.Doc.ExpiredDate != nil {
					nt.HasExpiry = true
				}
			}
			create.NewTypes = append(create.NewTypes, nt)
		case CVActionMatch:
			create.Aliases = append(create.Aliases, repository.CVAlias{Kind: r.Kind, Alias: cvAliasKey(c.Text), TargetID: r.ID})
		}
	}
	for _, rec := range plan.Records {
		var kind, newKey string
		var typeID uuid.UUID
		if rec.Match != nil {
			kind, typeID = rec.Match.Kind, rec.Match.ID
		} else {
			r := res[rec.ChoiceKey]
			switch r.Action {
			case CVActionMatch:
				kind, typeID = r.Kind, r.ID
			case CVActionCreate:
				kind, newKey = r.Kind, rec.ChoiceKey
			default:
				continue
			}
		}
		out := repository.CVRecord{TypeID: typeID, NewTypeKey: newKey, Doc: rec.Doc}
		if kind == repository.CVKindDocumentType {
			create.Documents = append(create.Documents, out)
		} else {
			create.Certificates = append(create.Certificates, out)
		}
	}
	for _, sp := range plan.Sea {
		x := sp.Experience
		if sp.RankKey != "" {
			r := res[sp.RankKey]
			if r.Action != CVActionMatch {
				continue
			}
			x.RankID = r.ID
		}
		var remarks []string
		if sp.Flag != "" {
			remarks = append(remarks, "Flag: "+sp.Flag)
		}
		var vesselKey string
		if sp.VesselKey != "" {
			switch r := res[sp.VesselKey]; r.Action {
			case CVActionMatch:
				id := r.ID
				x.VesselTypeID = &id
				remarks = append(remarks, "Vessel type in CV: "+sp.VesselText)
			case CVActionCreate:
				vesselKey = sp.VesselKey
			default:
				remarks = append(remarks, "Vessel type in CV: "+sp.VesselText)
			}
		}
		x.Remarks = cvClamp(strings.Join(remarks, "; "), 500)
		create.SeaService = append(create.SeaService, repository.CVSea{VesselTypeKey: vesselKey, Experience: x})
	}
	return create
}
