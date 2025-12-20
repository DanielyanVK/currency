package rates

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"service-currency/internal/models"
	"service-currency/internal/service/rates"
)

type Handler struct {
	rates *rates.Service
}

func New(r *rates.Service) *Handler { return &Handler{rates: r} }

func (h *Handler) Register(mux *nethttp.ServeMux) {
	mux.HandleFunc("/api/v1/rate", h.getRate)
}

func (h *Handler) getRate(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")

	out, err := h.rates.GetPairRate(r.Context(), base, quote)
	if err != nil {
		writeErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func writeErr(w nethttp.ResponseWriter, err error) {
	var be *models.BusinessError
	if errors.As(err, &be) {
		status := nethttp.StatusBadRequest
		if be.Code == "rate_not_available" {
			status = nethttp.StatusUnprocessableEntity
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)

		var resp models.BusinessError
		resp.Code = be.Code
		resp.Message = be.Message
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(nethttp.StatusInternalServerError)

	var resp models.BusinessError
	resp.Code = "internal"
	resp.Message = "internal error"
	_ = json.NewEncoder(w).Encode(resp)
}
