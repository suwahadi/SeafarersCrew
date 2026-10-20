package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/brevo"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// EmailSendJob is the job kind that delivers one email log.
const EmailSendJob = "email_send"

// Email limits.
const (
	EmailMaxRecipients      = 20
	EmailMaxAttachments     = 10
	EmailMaxAttachmentBytes = 10 << 20
	emailMaxAttempts        = 3
	emailMaxSubject         = 200
	emailMaxBody            = 100 << 10
)

// EmailRelatedTypes are the records an email can be logged against.
var EmailRelatedTypes = []string{"contract", "quotation", "crew", "generated_document", "applicant", "contact_message"}

// EmailStore persists templates and logs.
type EmailStore interface {
	ListTemplates(ctx context.Context) ([]model.EmailTemplate, error)
	GetTemplate(ctx context.Context, code string) (model.EmailTemplate, error)
	UpdateTemplate(ctx context.Context, code, subject, body string) (model.EmailTemplate, error)
	CreateLog(ctx context.Context, l model.EmailLog) (model.EmailLog, error)
	GetLog(ctx context.Context, id uuid.UUID) (model.EmailLog, error)
	ListLogs(ctx context.Context, f model.EmailLogFilter, limit, offset int) (repository.Page[model.EmailLog], error)
	MarkSent(ctx context.Context, id uuid.UUID, messageID string, at time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, msg string) error
	MarkRetrying(ctx context.Context, id uuid.UUID, msg string) error
	RequeueFailed(ctx context.Context, id uuid.UUID) (bool, error)
	LatestJobPayload(ctx context.Context, kind string, logID uuid.UUID) (json.RawMessage, error)
}

// EmailDelivery sends email through the provider.
type EmailDelivery interface {
	Send(ctx context.Context, m brevo.Message) (string, error)
	Account(ctx context.Context, apiKey string) (string, error)
}

// EmailAttachmentLoader reads a stored file for attaching to an email.
type EmailAttachmentLoader interface {
	Load(ctx context.Context, fileID uuid.UUID) (brevo.Attachment, error)
}

// EmailService renders, logs and dispatches email (PRD 6.7).
type EmailService struct {
	store       EmailStore
	delivery    EmailDelivery
	attachments EmailAttachmentLoader
	settings    SettingsReader
	jobs        Enqueuer
	policy      *bluemonday.Policy
	now         func() time.Time
}

// NewEmailService builds an EmailService.
func NewEmailService(store EmailStore, delivery EmailDelivery, attachments EmailAttachmentLoader, settings SettingsReader, jobs Enqueuer, now func() time.Time) *EmailService {
	return &EmailService{store: store, delivery: delivery, attachments: attachments, settings: settings, jobs: jobs, policy: bluemonday.UGCPolicy(), now: now}
}

// EmailSendInput is one email to queue.
type EmailSendInput struct {
	TemplateCode      string
	To                []string
	CC                []string
	Subject           string
	BodyHTML          string
	Variables         map[string]string
	RelatedType       string
	RelatedID         *uuid.UUID
	AttachmentFileIDs []uuid.UUID
}

type emailJobPayload struct {
	LogID    uuid.UUID `json:"log_id"`
	BodyHTML string    `json:"body_html"`
}

// Send validates and renders the email, logs it as queued and enqueues delivery.
func (s *EmailService) Send(ctx context.Context, in EmailSendInput) (model.EmailLog, model.Job, error) {
	to, cc, err := emailValidateInput(&in)
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	subjectTpl, bodyTpl := in.Subject, in.BodyHTML
	if in.TemplateCode != "" {
		t, err := s.template(ctx, in.TemplateCode)
		if err != nil {
			return model.EmailLog{}, model.Job{}, err
		}
		if subjectTpl == "" {
			subjectTpl = t.Subject
		}
		if bodyTpl == "" {
			bodyTpl = t.BodyHTML
		}
	}
	fields := map[string]string{}
	if strings.TrimSpace(subjectTpl) == "" {
		fields["subject"] = "A subject or a template is required."
	}
	if strings.TrimSpace(bodyTpl) == "" {
		fields["body_html"] = "A body or a template is required."
	}
	if len(fields) > 0 {
		return model.EmailLog{}, model.Job{}, apperr.Validation(fields)
	}
	vars, err := s.variables(ctx, in.Variables)
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	subject := RenderEmailSubject(subjectTpl, vars)
	body := s.policy.Sanitize(RenderEmail(bodyTpl, vars, true))
	log, err := s.store.CreateLog(ctx, model.EmailLog{
		TemplateCode: in.TemplateCode, To: to, CC: cc, Subject: subject, RelatedType: in.RelatedType,
		RelatedID: in.RelatedID, AttachmentFileIDs: in.AttachmentFileIDs,
	})
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	j, err := s.jobs.Enqueue(ctx, EmailSendJob, emailJobPayload{LogID: log.ID, BodyHTML: body})
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	return log, j, nil
}

// SendTemplate queues a template email to one recipient, e.g. the applicant confirmation.
func (s *EmailService) SendTemplate(ctx context.Context, code, to string, vars map[string]string, relatedType string, relatedID *uuid.UUID) (model.EmailLog, error) {
	log, _, err := s.Send(ctx, EmailSendInput{TemplateCode: code, To: []string{to}, Variables: vars, RelatedType: relatedType, RelatedID: relatedID})
	return log, err
}

// Notify queues a template email to the admin notification addresses; it does nothing when none are configured.
func (s *EmailService) Notify(ctx context.Context, code string, vars map[string]string, relatedType string, relatedID *uuid.UUID) error {
	values, err := s.settings.Values(ctx, "email")
	if err != nil {
		return err
	}
	to := emailStrings(values["admin_notification_emails"])
	if len(to) == 0 {
		return nil
	}
	_, _, err = s.Send(ctx, EmailSendInput{TemplateCode: code, To: to, Variables: vars, RelatedType: relatedType, RelatedID: relatedID})
	return err
}

// Resend queues a failed email again with its original body (PRD R7).
func (s *EmailService) Resend(ctx context.Context, id uuid.UUID) (model.EmailLog, model.Job, error) {
	log, err := s.store.GetLog(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.EmailLog{}, model.Job{}, apperr.NotFound("Email log")
	}
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	if log.Status != model.EmailFailed {
		return model.EmailLog{}, model.Job{}, apperr.Conflict("email_not_failed", "Only failed emails can be resent.")
	}
	payload, err := s.store.LatestJobPayload(ctx, EmailSendJob, id)
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	var p emailJobPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return model.EmailLog{}, model.Job{}, fmt.Errorf("decode email payload: %w", err)
	}
	ok, err := s.store.RequeueFailed(ctx, id)
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	if !ok {
		return model.EmailLog{}, model.Job{}, apperr.Conflict("email_not_failed", "Only failed emails can be resent.")
	}
	j, err := s.jobs.Enqueue(ctx, EmailSendJob, p)
	if err != nil {
		return model.EmailLog{}, model.Job{}, err
	}
	log.Status, log.Error = model.EmailQueued, ""
	return log, j, nil
}

// Handle delivers the email of an email_send job and records the outcome on its log.
func (s *EmailService) Handle(ctx context.Context, j model.Job) (any, error) {
	var p emailJobPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, job.Permanent{Err: fmt.Errorf("decode email payload: %w", err)}
	}
	log, err := s.store.GetLog(ctx, p.LogID)
	if err != nil {
		return nil, job.Permanent{Err: err}
	}
	if log.Status == model.EmailSent {
		return map[string]string{"brevo_message_id": log.BrevoMessageID}, nil
	}
	msg, err := s.message(ctx, log, p.BodyHTML)
	if err == nil {
		var id string
		if id, err = s.delivery.Send(ctx, msg); err == nil {
			if err := s.store.MarkSent(ctx, log.ID, id, s.now().UTC()); err != nil {
				return nil, err
			}
			return map[string]string{"brevo_message_id": id}, nil
		}
	}
	return nil, s.recordFailure(ctx, log.ID, j.Attempts, err)
}

func (s *EmailService) recordFailure(ctx context.Context, logID uuid.UUID, attempt int, sendErr error) error {
	var perm job.Permanent
	final := errors.As(sendErr, &perm) || !brevo.IsRetryable(sendErr) || attempt >= emailMaxAttempts
	if final {
		if err := s.store.MarkFailed(ctx, logID, sendErr.Error()); err != nil {
			return err
		}
		return job.Permanent{Err: sendErr}
	}
	if err := s.store.MarkRetrying(ctx, logID, sendErr.Error()); err != nil {
		return err
	}
	return sendErr
}

func (s *EmailService) message(ctx context.Context, log model.EmailLog, body string) (brevo.Message, error) {
	values, err := s.settings.Values(ctx, "email")
	if err != nil {
		return brevo.Message{}, err
	}
	str := func(k string) string {
		v, _ := values[k].(string)
		return v
	}
	if str("brevo_api_key") == "" || str("sender_email") == "" {
		return brevo.Message{}, job.Permanent{Err: errors.New("email is not configured: set the Brevo API key and sender email in Settings")}
	}
	m := brevo.Message{
		APIKey:  str("brevo_api_key"),
		Sender:  brevo.Address{Email: str("sender_email"), Name: str("sender_name")},
		To:      emailAddresses(log.To),
		CC:      emailAddresses(log.CC),
		Subject: log.Subject,
		HTML:    body,
	}
	if r := str("reply_to"); r != "" {
		m.ReplyTo = &brevo.Address{Email: r}
	}
	total := 0
	for _, id := range log.AttachmentFileIDs {
		a, err := s.attachments.Load(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return brevo.Message{}, job.Permanent{Err: fmt.Errorf("attachment %s no longer exists", id)}
		}
		if err != nil {
			return brevo.Message{}, err
		}
		total += len(a.Content)
		if total > EmailMaxAttachmentBytes {
			return brevo.Message{}, job.Permanent{Err: errors.New("attachments exceed 10 MB")}
		}
		m.Attachments = append(m.Attachments, a)
	}
	return m, nil
}

// ListLogs returns a page of email logs.
func (s *EmailService) ListLogs(ctx context.Context, f model.EmailLogFilter, page, perPage int) (repository.Page[model.EmailLog], error) {
	return s.store.ListLogs(ctx, f, perPage, (page-1)*perPage)
}

// Templates lists the email templates.
func (s *EmailService) Templates(ctx context.Context) ([]model.EmailTemplate, error) {
	return s.store.ListTemplates(ctx)
}

// Template returns one template.
func (s *EmailService) Template(ctx context.Context, code string) (model.EmailTemplate, error) {
	return s.template(ctx, code)
}

// UpdateTemplate validates, sanitizes and stores a template.
func (s *EmailService) UpdateTemplate(ctx context.Context, code, subject, body string) (model.EmailTemplate, error) {
	if _, err := s.template(ctx, code); err != nil {
		return model.EmailTemplate{}, err
	}
	subject = strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(subject))
	fields := map[string]string{}
	switch {
	case subject == "":
		fields["subject"] = "The subject is required."
	case utf8.RuneCountInString(subject) > emailMaxSubject:
		fields["subject"] = fmt.Sprintf("The subject may not exceed %d characters.", emailMaxSubject)
	}
	switch {
	case strings.TrimSpace(body) == "":
		fields["body_html"] = "The body is required."
	case len(body) > emailMaxBody:
		fields["body_html"] = "The body is too large."
	}
	if len(fields) > 0 {
		return model.EmailTemplate{}, apperr.Validation(fields)
	}
	return s.store.UpdateTemplate(ctx, code, subject, s.policy.Sanitize(body))
}

// EmailPreview is a rendered template.
type EmailPreview struct {
	Subject     string   `json:"subject"`
	BodyHTML    string   `json:"body_html"`
	UnknownTags []string `json:"unknown_tags"`
}

// PreviewTemplate renders a template, or unsaved subject and body, with sample values overridden by vars.
func (s *EmailService) PreviewTemplate(ctx context.Context, code, subject, body string, vars map[string]string) (EmailPreview, error) {
	t, err := s.template(ctx, code)
	if err != nil {
		return EmailPreview{}, err
	}
	if subject == "" {
		subject = t.Subject
	}
	if body == "" {
		body = t.BodyHTML
	}
	sample := EmailSampleVariables()
	for k, v := range vars {
		sample[k] = v
	}
	merged, err := s.variables(ctx, sample)
	if err != nil {
		return EmailPreview{}, err
	}
	known := EmailAvailableTags(code)
	unknown := []string{}
	for _, tag := range EmailTags(subject + " " + body) {
		if !slices.Contains(known, tag) {
			unknown = append(unknown, tag)
		}
	}
	return EmailPreview{Subject: RenderEmailSubject(subject, merged), BodyHTML: s.policy.Sanitize(RenderEmail(body, merged, true)), UnknownTags: unknown}, nil
}

func (s *EmailService) template(ctx context.Context, code string) (model.EmailTemplate, error) {
	t, err := s.store.GetTemplate(ctx, code)
	if errors.Is(err, repository.ErrNotFound) {
		return t, apperr.NotFound("Email template")
	}
	return t, err
}

func (s *EmailService) variables(ctx context.Context, vars map[string]string) (map[string]string, error) {
	company, err := s.settings.Values(ctx, "company")
	if err != nil {
		return nil, err
	}
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return nil, err
	}
	general, err := s.settings.Values(ctx, "general")
	if err != nil {
		return nil, err
	}
	pattern, _ := general["date_format"].(string)
	if pattern == "" {
		pattern = datetime.DefaultDisplayFormat
	}
	str := func(k string) string {
		v, _ := company[k].(string)
		return v
	}
	out := map[string]string{
		"company_name": str("legal_name"), "company_address": str("address"), "company_phone": str("phone"),
		"company_email": str("email"), "company_website": str("website"),
		"today": datetime.FormatDate(datetime.Today(s.now(), loc), pattern),
	}
	for k, v := range vars {
		out[k] = v
	}
	return out, nil
}

func emailValidateInput(in *EmailSendInput) ([]string, []string, error) {
	fields := map[string]string{}
	to, toErr := emailNormalizeAddresses(in.To)
	cc, ccErr := emailNormalizeAddresses(in.CC)
	switch {
	case toErr != "":
		fields["to"] = toErr
	case len(to) == 0:
		fields["to"] = "At least one recipient is required."
	}
	if ccErr != "" {
		fields["cc"] = ccErr
	}
	if len(to)+len(cc) > EmailMaxRecipients {
		fields["to"] = fmt.Sprintf("An email may have at most %d recipients.", EmailMaxRecipients)
	}
	if in.RelatedType != "" && !slices.Contains(EmailRelatedTypes, in.RelatedType) {
		fields["related_type"] = "Unknown related type."
	}
	if (in.RelatedType == "") != (in.RelatedID == nil) {
		fields["related_id"] = "Related type and related ID must be given together."
	}
	if len(in.AttachmentFileIDs) > EmailMaxAttachments {
		fields["attachment_file_ids"] = fmt.Sprintf("At most %d attachments are allowed.", EmailMaxAttachments)
	}
	if utf8.RuneCountInString(in.Subject) > emailMaxSubject {
		fields["subject"] = fmt.Sprintf("The subject may not exceed %d characters.", emailMaxSubject)
	}
	if len(in.BodyHTML) > emailMaxBody {
		fields["body_html"] = "The body is too large."
	}
	if len(fields) > 0 {
		return nil, nil, apperr.Validation(fields)
	}
	return to, cc, nil
}

func emailNormalizeAddresses(in []string) ([]string, string) {
	out := make([]string, 0, len(in))
	for _, raw := range in {
		addr := strings.ToLower(strings.TrimSpace(raw))
		if addr == "" {
			continue
		}
		parsed, err := mail.ParseAddress(addr)
		if err != nil || parsed.Address != addr || len(addr) > 254 {
			return nil, "Contains an invalid email address: " + raw
		}
		if !slices.Contains(out, addr) {
			out = append(out, addr)
		}
	}
	return out, ""
}

func emailAddresses(in []string) []brevo.Address {
	out := make([]brevo.Address, len(in))
	for i, a := range in {
		out[i] = brevo.Address{Email: a}
	}
	return out
}

func emailStrings(v any) []string {
	switch list := v.(type) {
	case []string:
		return list
	case []any:
		out := make([]string, 0, len(list))
		for _, it := range list {
			if s, ok := it.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
