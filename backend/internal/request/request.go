// Package request decodes and validates input payloads.
package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MaxJSONBody limits JSON request bodies.
const MaxJSONBody = 1 << 20

// Validator is implemented by request structs.
type Validator interface {
	Validate() map[string]string
}

// Decode reads a JSON body into dst and runs its validation.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	body := http.MaxBytesReader(w, r.Body, MaxJSONBody)
	dec := json.NewDecoder(body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return apperr.ErrPayloadTooLarge
		}
		if errors.Is(err, io.EOF) {
			return apperr.ErrBadRequest.Wrap(errors.New("empty body"))
		}
		return apperr.ErrBadRequest.Wrap(err)
	}
	if v, ok := dst.(Validator); ok {
		if fields := v.Validate(); len(fields) > 0 {
			return apperr.Validation(fields)
		}
	}
	return nil
}
