package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/team-service/internal/model"
	"github.com/nll-scheduler/team-service/internal/repository"
)

type Handler struct {
	divisions          *repository.DivisionRepo
	teams              *repository.TeamRepo
	matchupRules       *repository.MatchupRuleRepo
	divisionFieldRules *repository.DivisionFieldRuleRepo
}

func NewRouter(pool *pgxpool.Pool) *chi.Mux {
	h := &Handler{
		divisions:          repository.NewDivisionRepo(pool),
		teams:              repository.NewTeamRepo(pool),
		matchupRules:       repository.NewMatchupRuleRepo(pool),
		divisionFieldRules: repository.NewDivisionFieldRuleRepo(pool),
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.Health)
	r.Get("/export", h.ExportAll)
	r.Post("/import", h.ImportAll)

	r.Route("/divisions", func(r chi.Router) {
		r.Get("/", h.ListDivisions)
		r.Post("/", h.CreateDivision)
		r.Route("/{divisionID}", func(r chi.Router) {
			r.Get("/", h.GetDivision)
			r.Put("/", h.UpdateDivision)
			r.Delete("/", h.DeleteDivision)
			r.Get("/teams-with-rules", h.GetTeamsWithRules)
			r.Get("/field-rules", h.ListDivisionFieldRules)
			r.Post("/field-rules", h.CreateDivisionFieldRule)
			r.Delete("/field-rules/{ruleID}", h.DeleteDivisionFieldRule)
		})
	})

	r.Route("/teams", func(r chi.Router) {
		r.Get("/", h.ListTeams)
		r.Post("/", h.CreateTeam)
		r.Route("/{teamID}", func(r chi.Router) {
			r.Get("/", h.GetTeam)
			r.Put("/", h.UpdateTeam)
			r.Delete("/", h.DeleteTeam)
			r.Get("/matchup-rules", h.ListMatchupRules)
			r.Post("/matchup-rules", h.CreateMatchupRule)
			r.Delete("/matchup-rules/{ruleID}", h.DeleteMatchupRule)
		})
	})

	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "team-service"})
}

// Division handlers

func (h *Handler) ListDivisions(w http.ResponseWriter, r *http.Request) {
	divisions, err := h.divisions.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, divisions)
}

func (h *Handler) CreateDivision(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		SeasonYear int    `json:"season_year"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	d, err := h.divisions.Create(r.Context(), req.Name, req.SeasonYear)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) GetDivision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "divisionID")
	d, err := h.divisions.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "division not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) UpdateDivision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "divisionID")
	var req struct {
		Name       string `json:"name"`
		SeasonYear int    `json:"season_year"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	d, err := h.divisions.Update(r.Context(), id, req.Name, req.SeasonYear)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "division not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) DeleteDivision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "divisionID")
	if err := h.divisions.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "division not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetTeamsWithRules(w http.ResponseWriter, r *http.Request) {
	divisionID := chi.URLParam(r, "divisionID")
	result, err := h.teams.GetTeamsWithRules(r.Context(), divisionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Division field rule handlers

func (h *Handler) ListDivisionFieldRules(w http.ResponseWriter, r *http.Request) {
	divisionID := chi.URLParam(r, "divisionID")
	rules, err := h.divisionFieldRules.List(r.Context(), divisionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *Handler) CreateDivisionFieldRule(w http.ResponseWriter, r *http.Request) {
	divisionID := chi.URLParam(r, "divisionID")
	var req struct {
		FieldID  string `json:"field_id"`
		RuleType string `json:"rule_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.FieldID == "" {
		writeError(w, http.StatusBadRequest, "field_id is required")
		return
	}
	if req.RuleType != "allowed" && req.RuleType != "preferred" {
		writeError(w, http.StatusBadRequest, "rule_type must be 'allowed' or 'preferred'")
		return
	}
	rule, err := h.divisionFieldRules.Create(r.Context(), divisionID, req.FieldID, req.RuleType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteDivisionFieldRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "ruleID")
	if err := h.divisionFieldRules.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "rule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Team handlers

func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	divisionID := r.URL.Query().Get("division_id")
	teams, err := h.teams.List(r.Context(), divisionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, teams)
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req model.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.DivisionID == "" {
		writeError(w, http.StatusBadRequest, "name and division_id are required")
		return
	}
	t, err := h.teams.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "teamID")
	t, err := h.teams.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "teamID")
	var req model.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.teams.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "teamID")
	if err := h.teams.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Matchup rule handlers

func (h *Handler) ListMatchupRules(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	rules, err := h.matchupRules.List(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *Handler) CreateMatchupRule(w http.ResponseWriter, r *http.Request) {
	teamAID := chi.URLParam(r, "teamID")
	var req struct {
		TeamBID  string `json:"team_b_id"`
		MinGames int    `json:"min_games"`
		MaxGames int    `json:"max_games"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MinGames == 0 {
		req.MinGames = 1
	}
	if req.MaxGames == 0 {
		req.MaxGames = 3
	}
	rule, err := h.matchupRules.Create(r.Context(), teamAID, req.TeamBID, req.MinGames, req.MaxGames)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteMatchupRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "ruleID")
	if err := h.matchupRules.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "rule not found")
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
	divisions, err := h.divisions.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	teams, err := h.teams.List(ctx, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rules, err := h.matchupRules.ListAll(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	fieldRules, err := h.divisionFieldRules.ListAll(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"divisions":             divisions,
		"teams":                 teams,
		"matchup_rules":         rules,
		"division_field_rules":  fieldRules,
	})
}

func (h *Handler) ImportAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Divisions          []model.Division          `json:"divisions"`
		Teams              []model.Team              `json:"teams"`
		MatchupRules       []model.MatchupRule       `json:"matchup_rules"`
		DivisionFieldRules []model.DivisionFieldRule `json:"division_field_rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	for _, d := range req.Divisions {
		if err := h.divisions.Upsert(ctx, d); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, t := range req.Teams {
		if err := h.teams.Upsert(ctx, t); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, mr := range req.MatchupRules {
		if err := h.matchupRules.Upsert(ctx, mr); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, fr := range req.DivisionFieldRules {
		if err := h.divisionFieldRules.Upsert(ctx, fr); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imported_divisions":            len(req.Divisions),
		"imported_teams":                len(req.Teams),
		"imported_matchup_rules":        len(req.MatchupRules),
		"imported_division_field_rules": len(req.DivisionFieldRules),
	})
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
