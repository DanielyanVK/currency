package rates

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"service-currency/internal"
)

type Handler struct {
	rates  *internal.RateConverter
	logger internal.RequestAuditLogger
}

func New(r *internal.RateConverter, l internal.RequestAuditLogger) *Handler {
	return &Handler{rates: r, logger: l}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/rate", h.getRate)
}

func (h *Handler) getRate(w http.ResponseWriter, r *http.Request) {
	var err error

	if r.Method != http.MethodGet {
		st := http.StatusMethodNotAllowed

		writeErr(w, st, "method not allowed")

		err = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		if err != nil {
			log.Printf("audit log failed (path=%s status=%d): %v", r.URL.Path, st, err)
		}
		return
	}

	baseRaw := r.URL.Query().Get("base")
	quoteRaw := r.URL.Query().Get("quote")

	base, err := internal.NewCurrencyCode(baseRaw)
	if err != nil {
		st := http.StatusBadRequest
		writeErr(w, st, err.Error())
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	quote, err := internal.NewCurrencyCode(quoteRaw)
	if err != nil {
		st := http.StatusBadRequest
		writeErr(w, st, err.Error())
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	var out internal.PairRate
	out, err = h.rates.GetPairRate(r.Context(), base, quote)
	if err != nil {
		st := http.StatusBadRequest
		writeErr(w, st, err.Error())
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	out.Rate = out.Rate.Round(2)
	st := http.StatusOK
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(st)

	err = json.NewEncoder(w).Encode(out)
	if err != nil {
		log.Printf("encode response failed (path=%s status=%d): %v", r.URL.Path, st, err)
	}

	err = h.logger.LogRequest(r.Context(), r.URL.Path, &st, out.Date)
	if err != nil {
		log.Printf("audit log failed (path=%s status=%d): %v", r.URL.Path, st, err)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	var err error

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	err = json.NewEncoder(w).Encode(errors.New(msg).Error())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
