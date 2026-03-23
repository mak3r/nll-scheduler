package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
	"github.com/nll-scheduler/schedule-service/internal/repository"
)

// Generator orchestrates the full schedule generation workflow.
type Generator struct {
	db              *pgxpool.Pool
	teamClient      *TeamClient
	fieldClient     *FieldClient
	schedulerClient *SchedulerClient
	seasons         *repository.SeasonRepo
	extras          *repository.SeasonExtrasRepo
	games           *repository.GamesRepo
	genRuns         *repository.GenerationRunsRepo
}

func NewGenerator(
	db *pgxpool.Pool,
	teamServiceURL, fieldServiceURL, schedulerEngineURL string,
) *Generator {
	return &Generator{
		db:              db,
		teamClient:      NewTeamClient(teamServiceURL),
		fieldClient:     NewFieldClient(fieldServiceURL),
		schedulerClient: NewSchedulerClient(schedulerEngineURL),
		seasons:         repository.NewSeasonRepo(db),
		extras:          repository.NewSeasonExtrasRepo(db),
		games:           repository.NewGamesRepo(db),
		genRuns:         repository.NewGenerationRunsRepo(db),
	}
}

// GenerateAsync starts an async generation run.
// It immediately creates a GenerationRun record and launches a goroutine to do the work.
// Returns the run ID so the caller can poll status.
func (g *Generator) GenerateAsync(ctx context.Context, seasonID string) (string, error) {
	run, err := g.genRuns.Create(ctx, seasonID)
	if err != nil {
		return "", fmt.Errorf("create generation run: %w", err)
	}

	// Run in background — use a fresh context (caller's context may cancel)
	go func() {
		bgCtx := context.Background()
		if err := g.runGeneration(bgCtx, run.ID, seasonID); err != nil {
			log.Printf("generation run %s failed: %v", run.ID, err)
			errMsg := err.Error()
			if updateErr := g.genRuns.UpdateStatus(bgCtx, run.ID, "failed", nil, &errMsg); updateErr != nil {
				log.Printf("failed to update run status: %v", updateErr)
			}
			if updateErr := g.seasons.UpdateStatus(bgCtx, seasonID, "draft"); updateErr != nil {
				log.Printf("failed to reset season status: %v", updateErr)
			}
		}
	}()

	return run.ID, nil
}

func (g *Generator) runGeneration(ctx context.Context, runID, seasonID string) error {
	// Mark as running
	if err := g.genRuns.UpdateStatus(ctx, runID, "running", nil, nil); err != nil {
		return err
	}
	if err := g.seasons.UpdateStatus(ctx, seasonID, "generating"); err != nil {
		return err
	}

	// 1. Load season
	season, err := g.seasons.Get(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("load season: %w", err)
	}

	// 2. Fetch teams + matchup rules from team-service
	teamsWithRules, err := g.teamClient.GetTeamsWithRules(ctx, season.DivisionID)
	if err != nil {
		return fmt.Errorf("fetch teams: %w", err)
	}
	if len(teamsWithRules.Teams) < 2 {
		return fmt.Errorf("need at least 2 teams, got %d", len(teamsWithRules.Teams))
	}

	// 3. Fetch all active fields
	fields, err := g.fieldClient.ListFields(ctx)
	if err != nil {
		return fmt.Errorf("fetch fields: %w", err)
	}
	activeFields := make([]Field, 0, len(fields))
	fieldIDs := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.IsActive {
			activeFields = append(activeFields, f)
			fieldIDs = append(fieldIDs, f.ID)
		}
	}
	if len(fieldIDs) == 0 {
		return fmt.Errorf("no active fields available")
	}

	// 4. Fetch field availability for the season date range
	availMap, err := g.fieldClient.GetAvailableDatesBulk(ctx, fieldIDs, season.StartDate, season.EndDate)
	if err != nil {
		return fmt.Errorf("fetch field availability: %w", err)
	}

	// 5. Load season blackouts and preferred dates
	blackouts, err := g.extras.ListBlackouts(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("load blackouts: %w", err)
	}
	preferredDates, err := g.extras.ListPreferredDates(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("load preferred dates: %w", err)
	}
	constraints, err := g.extras.ListConstraints(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("load constraints: %w", err)
	}

	// 6. Build solver request
	solverTeams := make([]SolverTeam, len(teamsWithRules.Teams))
	for i, t := range teamsWithRules.Teams {
		solverTeams[i] = SolverTeam{
			ID:            t.ID,
			Name:          t.Name,
			DivisionID:    t.DivisionID,
			TeamType:      t.TeamType,
			GamesRequired: t.GamesRequired,
		}
	}

	solverMatchupRules := make([]SolverMatchupRule, len(teamsWithRules.MatchupRules))
	for i, r := range teamsWithRules.MatchupRules {
		solverMatchupRules[i] = SolverMatchupRule{
			TeamAID:  r.TeamAID,
			TeamBID:  r.TeamBID,
			MinGames: r.MinGames,
			MaxGames: r.MaxGames,
		}
	}

	solverFields := make([]SolverField, 0, len(activeFields))
	for _, f := range activeFields {
		slots := availMap[f.ID]
		solverSlots := make([]SolverFieldSlot, len(slots))
		for j, slot := range slots {
			solverSlots[j] = SolverFieldSlot{
				FieldID:   slot.FieldID,
				Date:      slot.Date,
				StartTime: slot.StartTime,
				EndTime:   slot.EndTime,
			}
		}
		solverFields = append(solverFields, SolverField{
			ID:             f.ID,
			Name:           f.Name,
			MaxGamesPerDay: f.MaxGamesPerDay,
			AvailableSlots: solverSlots,
		})
	}

	blackoutDates := make([]string, len(blackouts))
	for i, b := range blackouts {
		blackoutDates[i] = b.BlackoutDate
	}

	prefDates := make([]string, len(preferredDates))
	for i, p := range preferredDates {
		prefDates[i] = p.PreferredDate
	}

	solverConstraints := make([]SolverConstraint, len(constraints))
	for i, c := range constraints {
		params := c.Params
		if params == nil {
			params = json.RawMessage("{}")
		}
		solverConstraints[i] = SolverConstraint{
			Type:   c.Type,
			Params: params,
			IsHard: c.IsHard,
			Weight: c.Weight,
		}
	}

	// Compute games_per_pair from teams' games_required and inject into
	// round_robin_matchup unless the user has already set it explicitly.
	// Formula: round(avg_games_required / (n_teams - 1))
	nTeams := len(solverTeams)
	if nTeams >= 2 {
		totalRequired := 0
		for _, t := range solverTeams {
			totalRequired += t.GamesRequired
		}
		avgRequired := totalRequired / nTeams
		nPairs := nTeams - 1
		gamesPerPair := (avgRequired + nPairs/2) / nPairs // integer round
		if gamesPerPair < 1 {
			gamesPerPair = 1
		}

		// Find existing round_robin_matchup constraint
		rrIdx := -1
		for i, c := range solverConstraints {
			if c.Type == "round_robin_matchup" {
				rrIdx = i
				break
			}
		}
		if rrIdx >= 0 {
			// Inject default_games_per_pair only if not already explicitly set
			var p map[string]interface{}
			if err := json.Unmarshal(solverConstraints[rrIdx].Params, &p); err != nil || p == nil {
				p = map[string]interface{}{}
			}
			if _, alreadySet := p["default_games_per_pair"]; !alreadySet {
				p["default_games_per_pair"] = gamesPerPair
				updated, _ := json.Marshal(p)
				solverConstraints[rrIdx].Params = json.RawMessage(updated)
			}
		} else {
			// No round_robin_matchup configured; add one with computed games_per_pair
			params, _ := json.Marshal(map[string]int{"default_games_per_pair": gamesPerPair})
			solverConstraints = append(solverConstraints, SolverConstraint{
				Type:   "round_robin_matchup",
				Params: json.RawMessage(params),
				IsHard: true,
				Weight: 1.0,
			})
		}
		log.Printf("round_robin_matchup: games_per_pair=%d (avg_required=%d, n_teams=%d)", gamesPerPair, avgRequired, nTeams)

		// Pre-flight: check that max_games_per_team_per_week × season_weeks ≥ games_per_pair.
		// This catches the most common infeasibility before handing off to the solver.
		maxPerWeek := 2 // default (matches builder.py default)
		for _, c := range solverConstraints {
			if c.Type == "max_games_per_team_per_week" {
				var p map[string]interface{}
				if err := json.Unmarshal(c.Params, &p); err == nil {
					if v, ok := p["max_games_per_week"]; ok {
						switch n := v.(type) {
						case float64:
							maxPerWeek = int(n)
						case int:
							maxPerWeek = n
						}
					}
				}
				break
			}
		}
		seasonStart, _ := time.Parse("2006-01-02", season.StartDate)
		seasonEnd, _ := time.Parse("2006-01-02", season.EndDate)
		seasonWeeks := int(math.Ceil(seasonEnd.Sub(seasonStart).Hours() / (24 * 7)))
		maxAchievable := maxPerWeek * seasonWeeks
		if gamesPerPair > maxAchievable {
			return fmt.Errorf(
				"infeasible: need %d games per team but max_games_per_team_per_week=%d over %d weeks only allows %d — "+
					"extend the season, increase max games/week, or reduce games_required",
				gamesPerPair, maxPerWeek, seasonWeeks, maxAchievable,
			)
		}

		// Pre-flight: check total effective field slot capacity >= total games required.
		// Total games = C(n_teams, 2) * games_per_pair (each game involves 2 teams).
		totalGamesRequired := nTeams * (nTeams - 1) / 2 * gamesPerPair

		// Find explicit max_games_per_day override from constraints (if any)
		maxGamesPerDayOverride := 0
		for _, c := range solverConstraints {
			if c.Type == "max_games_per_field_per_day" {
				var p map[string]interface{}
				if err := json.Unmarshal(c.Params, &p); err == nil {
					if v, ok := p["max_games_per_day"]; ok {
						switch n := v.(type) {
						case float64:
							maxGamesPerDayOverride = int(n)
						case int:
							maxGamesPerDayOverride = n
						}
					}
				}
				break
			}
		}

		// Build season blackout set
		blackoutSet := make(map[string]bool, len(blackoutDates))
		for _, b := range blackoutDates {
			blackoutSet[b] = true
		}

		// Compute effective total capacity: each (field, date) contributes
		// min(slots_on_that_date, max_games_per_day) games.
		totalFieldCapacity := 0
		for _, field := range activeFields {
			slots := availMap[field.ID]
			slotsByDate := make(map[string]int)
			for _, slot := range slots {
				if !blackoutSet[slot.Date] {
					slotsByDate[slot.Date]++
				}
			}
			limit := field.MaxGamesPerDay
			if maxGamesPerDayOverride > 0 {
				limit = maxGamesPerDayOverride
			}
			if limit <= 0 {
				limit = 4
			}
			for _, count := range slotsByDate {
				if count > limit {
					count = limit
				}
				totalFieldCapacity += count
			}
		}

		if totalGamesRequired > totalFieldCapacity {
			return fmt.Errorf(
				"infeasible: need %d game slots (%d pairs × %d games/pair) but fields only provide %d effective slots after applying max_games_per_day limits — "+
					"add more fields or availability windows, increase max_games_per_day, or reduce games_required",
				totalGamesRequired, nTeams*(nTeams-1)/2, gamesPerPair, totalFieldCapacity,
			)
		}
	}

	solveReq := SolveRequest{
		SeasonID:                  seasonID,
		StartDate:                 season.StartDate,
		EndDate:                   season.EndDate,
		Teams:                     solverTeams,
		MatchupRules:              solverMatchupRules,
		Fields:                    solverFields,
		BlackoutDates:             blackoutDates,
		PreferredInterleagueDates: prefDates,
		Constraints:               solverConstraints,
		TimeLimitSeconds:          60,
	}

	// 7. Call scheduler-engine
	log.Printf("Calling scheduler-engine for season %s (%d teams, %d fields)", seasonID, len(solverTeams), len(solverFields))
	solveResp, err := g.schedulerClient.Solve(ctx, solveReq)
	if err != nil {
		return fmt.Errorf("solver error: %w", err)
	}

	if solveResp.Status == "infeasible" {
		statsJSON, _ := json.Marshal(solveResp.SolverStats)
		errMsg := "solver returned infeasible: " + strings.Join(solveResp.UnmetConstraints, "; ")
		if err := g.genRuns.UpdateStatus(ctx, runID, "failed", statsJSON, &errMsg); err != nil {
			return err
		}
		return g.seasons.UpdateStatus(ctx, seasonID, "draft")
	}

	// 8. Delete old games for this season and persist new ones
	if err := g.games.DeleteBySeason(ctx, seasonID); err != nil {
		return fmt.Errorf("clear old games: %w", err)
	}

	newGames := make([]model.Game, len(solveResp.Games))
	for i, sg := range solveResp.Games {
		newGames[i] = model.Game{
			SeasonID:      seasonID,
			HomeTeamID:    sg.HomeTeamID,
			AwayTeamID:    sg.AwayTeamID,
			FieldID:       sg.FieldID,
			GameDate:      sg.GameDate,
			StartTime:     sg.StartTime,
			Status:        "scheduled",
			IsInterleague: sg.IsInterleague,
		}
	}

	if err := g.games.BulkCreate(ctx, newGames); err != nil {
		return fmt.Errorf("persist games: %w", err)
	}

	// 9. Mark run as success
	statsJSON, _ := json.Marshal(solveResp.SolverStats)
	if err := g.genRuns.UpdateStatus(ctx, runID, "success", statsJSON, nil); err != nil {
		return err
	}
	if err := g.seasons.UpdateStatus(ctx, seasonID, "review"); err != nil {
		return err
	}

	log.Printf("Generation complete for season %s: %d games scheduled (status=%s)", seasonID, len(newGames), solveResp.Status)
	return nil
}
