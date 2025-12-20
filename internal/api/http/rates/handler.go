package rates

import (
	"encoding/json"
	"net/http"
	"service-currency/internal/models"
	"service-currency/internal/service/logger"
	"service-currency/internal/service/rates"
)

type Handler struct {
	rates  *rates.Service
	logger logger.RequestLogger
}

func New(r *rates.Service, l logger.RequestLogger) *Handler {
	return &Handler{rates: r, logger: l}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/rate", h.getRate)
}

func (h *Handler) getRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		st := http.StatusMethodNotAllowed
		w.WriteHeader(st)
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")

	out, err := h.rates.GetPairRate(r.Context(), base, quote)
	if err != nil {
		st := writeErr(w, err)
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	st := http.StatusOK
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
	_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, out.Date)
}

// Наивная обработка ошибок
func writeErr(w http.ResponseWriter, err error) int {
	status := http.StatusBadRequest

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(models.BusinessError{
		Code:    "bad_request",
		Message: err.Error(),
	})

	return status
}
