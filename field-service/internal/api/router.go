package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/field-service/internal/domain"
	"github.com/nll-scheduler/field-service/internal/model"
	"github.com/nll-scheduler/field-service/internal/repository"
)

type Handler struct {
	fields       *repository.FieldRepo
	availability *repository.AvailabilityRepo
}

func NewRouter(pool *pgxpool.Pool) *chi.Mux {
	h := &Handler{
		fields:       repository.NewFieldRepo(pool),
		availability: repository.NewAvailabilityRepo(pool),
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.Health)
	r.Get("/export", h.ExportAll)
	r.Post("/import", h.ImportAll)

	// Bulk availability endpoint (must be before /{fieldID} to avoid route conflict)
	r.Get("/fields/available-dates-bulk", h.GetAvailableDatesBulk)

	r.Route("/fields", func(r chi.Router) {
		r.Get("/", h.ListFields)
		r.Post("/", h.CreateField)
		r.Route("/{fieldID}", func(r chi.Router) {
			r.Get("/", h.GetField)
			r.Put("/", h.UpdateField)
			r.Delete("/", h.DeleteField)
			r.Route("/availability-windows", func(r chi.Router) {
				r.Get("/", h.ListAvailabilityWindows)
				r.Post("/", h.CreateAvailabilityWindow)
				r.Put("/{windowID}", h.UpdateAvailabilityWindow)
				r.Delete("/{windowID}", h.DeleteAvailabilityWindow)
			})
			r.Route("/blackout-dates", func(r chi.Router) {
				r.Get("/", h.ListBlackoutDates)
				r.Post("/", h.CreateBlackoutDate)
				r.Delete("/{blackoutID}", h.DeleteBlackoutDate)
			})
		})
	})

	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "field-service"})
}

func (h *Handler) GetAvailableDatesBulk(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	fieldIDsStr := r.URL.Query().Get("field_ids")

	if startStr == "" || endStr == "" || fieldIDsStr == "" {
		writeError(w, http.StatusBadRequest, "start, end, and field_ids are required")
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid start date format, use YYYY-MM-DD")
		return
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid end date format, use YYYY-MM-DD")
		return
	}

	fieldIDs := strings.Split(fieldIDsStr, ",")
	result := make(map[string][]model.AvailableSlot)

	for _, fieldID := range fieldIDs {
		fieldID = strings.TrimSpace(fieldID)
		if fieldID == "" {
			continue
		}
		windows, err := h.availability.ListWindows(r.Context(), fieldID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		blackouts, err := h.availability.ListBlackouts(r.Context(), fieldID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		slots := domain.MaterializeSlots(fieldID, windows, blackouts, start, end)
		if slots == nil {
			slots = []model.AvailableSlot{}
		}
		result[fieldID] = slots
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListFields(w http.ResponseWriter, r *http.Request) {
	fields, err := h.fields.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, fields)
}

func (h *Handler) CreateField(w http.ResponseWriter, r *http.Request) {
	var req model.Field
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.MaxGamesPerDay == 0 {
		req.MaxGamesPerDay = 4
	}
	req.IsActive = true
	f, err := h.fields.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) GetField(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "fieldID")
	f, err := h.fields.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "field not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) UpdateField(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "fieldID")
	var req model.Field
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	f, err := h.fields.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "field not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) DeleteField(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "fieldID")
	if err := h.fields.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "field not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListAvailabilityWindows(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "fieldID")
	windows, err := h.availability.ListWindows(r.Context(), fieldID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, windows)
}

func (h *Handler) CreateAvailabilityWindow(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "fieldID")
	var req model.AvailabilityWindow
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.FieldID = fieldID
	w2, err := h.availability.CreateWindow(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, w2)
}

func (h *Handler) UpdateAvailabilityWindow(w http.ResponseWriter, r *http.Request) {
	windowID := chi.URLParam(r, "windowID")
	fieldID := chi.URLParam(r, "fieldID")
	var req model.AvailabilityWindow
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ID = windowID
	req.FieldID = fieldID
	updated, err := h.availability.UpdateWindow(r.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "window not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteAvailabilityWindow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "windowID")
	if err := h.availability.DeleteWindow(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "window not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListBlackoutDates(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "fieldID")
	blackouts, err := h.availability.ListBlackouts(r.Context(), fieldID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, blackouts)
}

func (h *Handler) CreateBlackoutDate(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "fieldID")
	var req model.BlackoutDate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.FieldID = fieldID
	b, err := h.availability.CreateBlackout(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handler) DeleteBlackoutDate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "blackoutID")
	if err := h.availability.DeleteBlackout(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "blackout not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Export/Import handlers

func (h *Handler) ExportAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fields, err := h.fields.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	windows, err := h.availability.ListAllWindows(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	blackouts, err := h.availability.ListAllBlackouts(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"fields":               fields,
		"availability_windows": windows,
		"blackout_dates":       blackouts,
	})
}

func (h *Handler) ImportAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Fields              []model.Field              `json:"fields"`
		AvailabilityWindows []model.AvailabilityWindow `json:"availability_windows"`
		BlackoutDates       []model.BlackoutDate       `json:"blackout_dates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	for _, f := range req.Fields {
		if err := h.fields.Upsert(ctx, f); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, win := range req.AvailabilityWindows {
		if err := h.availability.UpsertWindow(ctx, win); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, b := range req.BlackoutDates {
		if err := h.availability.UpsertBlackout(ctx, b); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imported_fields":               len(req.Fields),
		"imported_availability_windows": len(req.AvailabilityWindows),
		"imported_blackout_dates":       len(req.BlackoutDates),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
