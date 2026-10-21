package public

import (
	"context"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// ContactService is what the contact form controller needs.
type ContactService interface {
	Submit(ctx context.Context, in service.ContactInput) error
}

// ContactHandler receives contact form messages.
type ContactHandler struct {
	contact ContactService
}

// NewContactHandler returns a ContactHandler.
func NewContactHandler(contact ContactService) *ContactHandler {
	return &ContactHandler{contact: contact}
}

type contactBody struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Subject        string `json:"subject"`
	Message        string `json:"message"`
	Website        string `json:"website"`
	TurnstileToken string `json:"cf-turnstile-response"`
}

// Store handles POST /contact.
func (h *ContactHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in contactBody
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	err := h.contact.Submit(r.Context(), service.ContactInput{
		Name: in.Name, Email: in.Email, Phone: in.Phone, Subject: in.Subject, Message: in.Message,
		Website: in.Website, TurnstileToken: in.TurnstileToken, IP: middleware.ClientIP(r.Context()),
	})
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, map[string]bool{"received": true})
}
