package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
	"github.com/nll-scheduler/schedule-service/internal/orchestrator"
	"github.com/nll-scheduler/schedule-service/internal/repository"
)

type Handler struct {
	seasons   *repository.SeasonRepo
	extras    *repository.SeasonExtrasRepo
	games     *repository.GamesRepo
	genRuns   *repository.GenerationRunsRepo
	generator *orchestrator.Generator
}

func NewRouter(pool *pgxpool.Pool, teamServiceURL, fieldServiceURL, schedulerEngineURL string) *chi.Mux {
	h := &Handler{
		seasons:   repository.NewSeasonRepo(pool),
		extras:    repository.NewSeasonExtrasRepo(pool),
		games:     repository.NewGamesRepo(pool),
		genRuns:   repository.NewGenerationRunsRepo(pool),
		generator: orchestrator.NewGenerator(pool, teamServiceURL, fieldServiceURL, schedulerEngineURL),
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.Health)
	r.Get("/export", h.ExportAll)
	r.Post("/import", h.ImportAll)

	r.Route("/seasons", func(r chi.Router) {
		r.Get("/", h.ListSeasons)
		r.Post("/", h.CreateSeason)
		r.Route("/{seasonID}", func(r chi.Router) {
			r.Get("/", h.GetSeason)
			r.Put("/", h.UpdateSeason)
			r.Delete("/", h.DeleteSeason)

			r.Route("/blackout-dates", func(r chi.Router) {
				r.Get("/", h.ListSeasonBlackouts)
				r.Post("/", h.CreateSeasonBlackout)
				r.Delete("/{blackoutID}", h.DeleteSeasonBlackout)
			})

			r.Route("/preferred-interleague-dates", func(r chi.Router) {
				r.Get("/", h.ListPreferredDates)
				r.Post("/", h.CreatePreferredDate)
				r.Delete("/{prefID}", h.DeletePreferredDate)
			})

			r.Route("/constraints", func(r chi.Router) {
				r.Get("/", h.ListConstraints)
				r.Post("/", h.CreateConstraint)
				r.Put("/{constraintID}", h.UpdateConstraint)
				r.Delete("/{constraintID}", h.DeleteConstraint)
			})

			r.Route("/games", func(r chi.Router) {
				r.Get("/", h.ListGames)
				r.Post("/", h.CreateGame)
				r.Post("/check-conflicts", h.CheckConflicts)
				r.Get("/summary", h.GamesSummary)
				r.Route("/{gameID}", func(r chi.Router) {
					r.Get("/", h.GetGame)
					r.Put("/", h.UpdateGame)
					r.Delete("/", h.DeleteGame)
				})
			})

			r.Post("/generate", h.GenerateSchedule)
			r.Get("/generate/{runID}", h.GetGenerationRun)
			r.Get("/export", h.ExportSchedule)
		})
	})

	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "schedule-service"})
}

// Season handlers

func (h *Handler) ListSeasons(w http.ResponseWriter, r *http.Request) {
	seasons, err := h.seasons.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, seasons)
}

func (h *Handler) CreateSeason(w http.ResponseWriter, r *http.Request) {
	var req model.Season
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.DivisionIDs) == 0 {
		writeError(w, http.StatusBadRequest, "season must include at least one division")
		return
	}
	s, err := h.seasons.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) GetSeason(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "seasonID")
	s, err := h.seasons.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "season not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) UpdateSeason(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "seasonID")
	var req model.Season
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	s, err := h.seasons.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "season not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) DeleteSeason(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "seasonID")
	if err := h.seasons.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "season not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Blackout handlers

func (h *Handler) ListSeasonBlackouts(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	blackouts, err := h.extras.ListBlackouts(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, blackouts)
}

func (h *Handler) CreateSeasonBlackout(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	var req struct {
		BlackoutDate string `json:"blackout_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	b, err := h.extras.CreateBlackout(r.Context(), seasonID, req.BlackoutDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handler) DeleteSeasonBlackout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "blackoutID")
	if err := h.extras.DeleteBlackout(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "blackout not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Preferred date handlers

func (h *Handler) ListPreferredDates(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	dates, err := h.extras.ListPreferredDates(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dates)
}

func (h *Handler) CreatePreferredDate(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	var req struct {
		PreferredDate string  `json:"preferred_date"`
		Weight        float64 `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Weight == 0 {
		req.Weight = 1.0
	}
	d, err := h.extras.CreatePreferredDate(r.Context(), seasonID, req.PreferredDate, req.Weight)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) DeletePreferredDate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "prefID")
	if err := h.extras.DeletePreferredDate(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "preferred date not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Constraint handlers

func (h *Handler) ListConstraints(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	constraints, err := h.extras.ListConstraints(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, constraints)
}

func (h *Handler) CreateConstraint(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	var req model.SeasonConstraint
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.SeasonID = seasonID
	if req.Params == nil {
		req.Params = json.RawMessage("{}")
	}
	if req.Weight == 0 {
		req.Weight = 1.0
	}
	c, err := h.extras.CreateConstraint(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateConstraint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "constraintID")
	var req model.SeasonConstraint
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.extras.UpdateConstraint(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "constraint not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteConstraint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "constraintID")
	if err := h.extras.DeleteConstraint(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "constraint not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Game handlers

func (h *Handler) ListGames(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	games, err := h.games.List(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, games)
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	var req model.Game
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.SeasonID = seasonID
	if req.Status == "" {
		req.Status = "scheduled"
	}
	g, err := h.games.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "gameID")
	g, err := h.games.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *Handler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "gameID")
	var req model.Game
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	g, err := h.games.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *Handler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "gameID")
	if err := h.games.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CheckConflicts(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	games, err := h.games.List(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	conflicts := detectConflicts(games)
	writeJSON(w, http.StatusOK, map[string]any{"conflicts": conflicts})
}

// detectConflicts finds games where the same field is double-booked at the same date/time.
func detectConflicts(games []model.Game) []string {
	type key struct{ fieldID, date, startTime string }
	seen := make(map[key]string) // key -> game ID
	var conflicts []string
	for _, g := range games {
		k := key{g.FieldID, g.GameDate, g.StartTime}
		if existing, ok := seen[k]; ok {
			conflicts = append(conflicts,
				"Field double-booked on "+g.GameDate+" at "+g.StartTime+
					": games "+existing+" and "+g.ID)
		} else {
			seen[k] = g.ID
		}
	}
	if conflicts == nil {
		conflicts = []string{}
	}
	return conflicts
}

func (h *Handler) GenerateSchedule(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	runID, err := h.generator.GenerateAsync(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"run_id": runID})
}

func (h *Handler) GetGenerationRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	run, err := h.genRuns.Get(r.Context(), runID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) GamesSummary(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	stats, err := h.games.SummaryBySeason(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Group by division
	type teamEntry struct {
		TeamID string `json:"team_id"`
		Home   int    `json:"home"`
		Away   int    `json:"away"`
		Total  int    `json:"total"`
	}
	type divEntry struct {
		DivisionID string      `json:"division_id"`
		Teams      []teamEntry `json:"teams"`
	}

	divMap := make(map[string]*divEntry)
	divOrder := make([]string, 0)
	for _, s := range stats {
		if _, ok := divMap[s.DivisionID]; !ok {
			divMap[s.DivisionID] = &divEntry{DivisionID: s.DivisionID, Teams: []teamEntry{}}
			divOrder = append(divOrder, s.DivisionID)
		}
		divMap[s.DivisionID].Teams = append(divMap[s.DivisionID].Teams, teamEntry{
			TeamID: s.TeamID,
			Home:   s.Home,
			Away:   s.Away,
			Total:  s.Home + s.Away,
		})
	}

	result := make([]divEntry, 0, len(divOrder))
	for _, divID := range divOrder {
		result = append(result, *divMap[divID])
	}
	writeJSON(w, http.StatusOK, map[string]any{"divisions": result})
}

func (h *Handler) ExportSchedule(w http.ResponseWriter, r *http.Request) {
	seasonID := chi.URLParam(r, "seasonID")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" {
		writeError(w, http.StatusNotImplemented, "only json format currently supported")
		return
	}
	games, err := h.games.List(r.Context(), seasonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"format": "json", "games": games})
}

// seasonWithExtras is the transport type for full export/import of a season with all sub-resources.
type seasonWithExtras struct {
	model.Season
	BlackoutDates  []model.SeasonBlackout   `json:"blackout_dates"`
	PreferredDates []model.PreferredDate    `json:"preferred_interleague_dates"`
	Constraints    []model.SeasonConstraint `json:"constraints"`
	Games          []model.Game             `json:"games"`
}

// Export/Import handlers

func (h *Handler) ExportAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	seasons, err := h.seasons.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	allBlackouts, err := h.extras.ListAllBlackouts(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	allPreferred, err := h.extras.ListAllPreferredDates(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	allConstraints, err := h.extras.ListAllConstraints(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	allGames, err := h.games.ListAll(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Group sub-resources by season_id.
	blackoutsBySeasonID := make(map[string][]model.SeasonBlackout)
	for _, b := range allBlackouts {
		blackoutsBySeasonID[b.SeasonID] = append(blackoutsBySeasonID[b.SeasonID], b)
	}
	preferredBySeasonID := make(map[string][]model.PreferredDate)
	for _, d := range allPreferred {
		preferredBySeasonID[d.SeasonID] = append(preferredBySeasonID[d.SeasonID], d)
	}
	constraintsBySeasonID := make(map[string][]model.SeasonConstraint)
	for _, c := range allConstraints {
		constraintsBySeasonID[c.SeasonID] = append(constraintsBySeasonID[c.SeasonID], c)
	}
	gamesBySeasonID := make(map[string][]model.Game)
	for _, g := range allGames {
		gamesBySeasonID[g.SeasonID] = append(gamesBySeasonID[g.SeasonID], g)
	}

	result := make([]seasonWithExtras, len(seasons))
	for i, s := range seasons {
		blackouts := blackoutsBySeasonID[s.ID]
		if blackouts == nil {
			blackouts = []model.SeasonBlackout{}
		}
		preferred := preferredBySeasonID[s.ID]
		if preferred == nil {
			preferred = []model.PreferredDate{}
		}
		constraints := constraintsBySeasonID[s.ID]
		if constraints == nil {
			constraints = []model.SeasonConstraint{}
		}
		games := gamesBySeasonID[s.ID]
		if games == nil {
			games = []model.Game{}
		}
		result[i] = seasonWithExtras{
			Season:         s,
			BlackoutDates:  blackouts,
			PreferredDates: preferred,
			Constraints:    constraints,
			Games:          games,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"seasons": result})
}

func (h *Handler) ImportAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Seasons []seasonWithExtras `json:"seasons"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	totalGames := 0
	for _, s := range req.Seasons {
		if err := h.seasons.Upsert(ctx, s.Season); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, b := range s.BlackoutDates {
			if err := h.extras.UpsertBlackout(ctx, b); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		for _, d := range s.PreferredDates {
			if err := h.extras.UpsertPreferredDate(ctx, d); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		for _, c := range s.Constraints {
			if err := h.extras.UpsertConstraint(ctx, c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		for _, g := range s.Games {
			if err := h.games.Upsert(ctx, g); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		totalGames += len(s.Games)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imported_seasons": len(req.Seasons),
		"imported_games":   totalGames,
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
