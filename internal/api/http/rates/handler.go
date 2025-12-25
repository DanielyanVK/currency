package rates

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"service-currency/internal"
)

type Handler struct {
	rates               *internal.RateConverter
	logger              internal.RequestAuditLogger
	client              internal.RatesClient
	supportedCurrencies []internal.CurrencyCode
}

func New(
	rates *internal.RateConverter,
	logger internal.RequestAuditLogger,
	client internal.RatesClient,
	supportedCurrencies []internal.CurrencyCode,
) *Handler {
	return &Handler{rates: rates, logger: logger, client: client, supportedCurrencies: supportedCurrencies}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/rate", h.getRate)
	mux.HandleFunc("/api/v1/rate/historical", h.getHistoricalRates)
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

func (h *Handler) getHistoricalRates(w http.ResponseWriter, r *http.Request) {
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

	dateRaw := r.URL.Query().Get("date")
	baseRaw := r.URL.Query().Get("base")

	if dateRaw == "" {
		st := http.StatusBadRequest
		writeErr(w, st, "date parameter is required")
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	if baseRaw == "" {
		st := http.StatusBadRequest
		writeErr(w, st, "base parameter is required")
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	parsedTime, err := time.Parse("2006-01-02", dateRaw)
	if err != nil {
		st := http.StatusBadRequest
		writeErr(w, st, "invalid date format, expected YYYY-MM-DD")
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}
	date := internal.Date{Time: parsedTime}

	base, err := internal.NewCurrencyCode(baseRaw)
	if err != nil {
		st := http.StatusBadRequest
		writeErr(w, st, err.Error())
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	symbols := make([]internal.CurrencyCode, 0, len(h.supportedCurrencies))
	for _, ccy := range h.supportedCurrencies {
		if ccy != base {
			symbols = append(symbols, ccy)
		}
	}

	historicalResp, err := h.client.HistoricalRates(r.Context(), date, base, symbols)
	if err != nil {
		st := http.StatusBadRequest
		writeErr(w, st, err.Error())
		_ = h.logger.LogRequest(r.Context(), r.URL.Path, &st, nil)
		return
	}

	st := http.StatusOK
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(st)

	err = json.NewEncoder(w).Encode(historicalResp)
	if err != nil {
		log.Printf("encode response failed (path=%s status=%d): %v", r.URL.Path, st, err)
	}

	err = h.logger.LogRequest(r.Context(), r.URL.Path, &st, &historicalResp.Date)
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
