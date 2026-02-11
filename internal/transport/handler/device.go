package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/alonsoF100/reporting-service/internal/models"
	"github.com/go-chi/chi/v5"
)

type Service interface {
	GetDeviceMessages(ctx context.Context, unitGUID string, page, limit int) ([]models.DeviceMessage, int, error)
}

type Handler struct {
	Service Service
}

func New(service Service) *Handler {
	return &Handler{
		Service: service,
	}
}

/*
pattern: /api/v1/devices/{id}
method: GET
query: page, limit
info: Get paginated messages for device by unit_guid

succeed:
  - status code: 200 OK
  - response body: JSON with messages and pagination info

failed:
  - status code: 400 bad request - invalid parameters
  - status code: 404 not found - device not found
  - status code: 500 internal server error
  - response body: JSON with error message
*/
func (h *Handler) GetDeviceMessages(w http.ResponseWriter, r *http.Request) {
	unitGUID := chi.URLParam(r, "id")
	if unitGUID == "" {
		respondWithError(w, http.StatusBadRequest, "unit_guid is required")
		return
	}

	page := parseInt(r.URL.Query().Get("page"), 1)
	limit := parseInt(r.URL.Query().Get("limit"), 50)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	messages, total, err := h.Service.GetDeviceMessages(r.Context(), unitGUID, page, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(messages) == 0 {
		respondWithError(w, http.StatusNotFound, "device not found or no messages")
		return
	}

	// Формируем ответ
	response := struct {
		UnitGUID string                 `json:"unit_guid"`
		Invid    string                 `json:"invid"`
		Total    int                    `json:"total"`
		Page     int                    `json:"page"`
		Limit    int                    `json:"limit"`
		Pages    int                    `json:"pages"`
		Messages []models.DeviceMessage `json:"messages"`
	}{
		UnitGUID: unitGUID,
		Invid:    messages[0].Invid,
		Total:    total,
		Page:     page,
		Limit:    limit,
		Pages:    (total + limit - 1) / limit,
		Messages: messages,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		}
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
