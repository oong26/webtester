package response

import (
	"encoding/json"
	"net/http"

	apperr "webtester/pkg/errors"
)

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func Fail(w http.ResponseWriter, err error) {
	ae, ok := err.(*apperr.AppError)
	if !ok {
		ae = apperr.Internal("unexpected error", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ae.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    ae.Code,
			"message": ae.Message,
		},
	})
}
