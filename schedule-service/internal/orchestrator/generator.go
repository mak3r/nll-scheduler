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
	if len(season.DivisionIDs) == 0 {
		return fmt.Errorf("season has no divisions configured")
	}

	// 2. Fetch division names for human-readable error messages, then
	//    fetch teams + matchup rules from ALL divisions.
	divisionNames := make(map[string]string) // divID → name
	for _, divID := range season.DivisionIDs {
		div, err := g.teamClient.GetDivision(ctx, divID)
		if err != nil {
			// Non-fatal: fall back to ID in messages
			divisionNames[divID] = divID
		} else {
			divisionNames[divID] = div.Name
		}
	}

	// divLabel returns "Name (id)" for error messages.
	divLabel := func(divID string) string {
		name, ok := divisionNames[divID]
		if !ok || name == divID {
			return divID
		}
		return fmt.Sprintf("%s (%s)", name, divID)
	}

	allTeams := make([]Team, 0)
	allMatchupRules := make([]MatchupRule, 0)
	divisionFieldRestrictions := make(map[string][]string) // divID → allowed field IDs (omit = all)
	divisionPreferredFields := make(map[string][]string)   // divID → preferred field IDs

	for _, divID := range season.DivisionIDs {
		twr, err := g.teamClient.GetTeamsWithRules(ctx, divID)
		if err != nil {
			return fmt.Errorf("fetch teams for division %s: %w", divLabel(divID), err)
		}
		allTeams = append(allTeams, twr.Teams...)
		allMatchupRules = append(allMatchupRules, twr.MatchupRules...)

		fieldRules, err := g.teamClient.GetDivisionFieldRules(ctx, divID)
		if err != nil {
			return fmt.Errorf("fetch field rules for division %s: %w", divLabel(divID), err)
		}
		for _, r := range fieldRules {
			if r.RuleType == "allowed" {
				divisionFieldRestrictions[divID] = append(divisionFieldRestrictions[divID], r.FieldID)
			}
			if r.RuleType == "preferred" {
				divisionPreferredFields[divID] = append(divisionPreferredFields[divID], r.FieldID)
			}
		}
		// Division with no "allowed" rules → omit from divisionFieldRestrictions (solver treats as "all fields")
		// "preferred" rules only → soft preference, no hard restriction
	}

	if len(allTeams) < 2 {
		return fmt.Errorf("need at least 2 teams across all divisions, got %d", len(allTeams))
	}

	// 3. Fetch all active fields, including only those accessible by at least one division
	fields, err := g.fieldClient.ListFields(ctx)
	if err != nil {
		return fmt.Errorf("fetch fields: %w", err)
	}

	activeFields := make([]Field, 0, len(fields))
	fieldIDs := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.IsActive {
			continue
		}
		// Include field if any division can use it
		accessible := false
		for _, divID := range season.DivisionIDs {
			allowed, hasRules := divisionFieldRestrictions[divID]
			if !hasRules {
				accessible = true // Division with no restrictions uses all fields
				break
			}
			for _, fid := range allowed {
				if fid == f.ID {
					accessible = true
					break
				}
			}
			if accessible {
				break
			}
		}
		if accessible {
			activeFields = append(activeFields, f)
			fieldIDs = append(fieldIDs, f.ID)
		}
	}
	if len(fieldIDs) == 0 {
		return fmt.Errorf("no active fields available — check field access rules")
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
	solverTeams := make([]SolverTeam, len(allTeams))
	for i, t := range allTeams {
		solverTeams[i] = SolverTeam{
			ID:            t.ID,
			Name:          t.Name,
			DivisionID:    t.DivisionID,
			TeamType:      t.TeamType,
			GamesRequired: t.GamesRequired,
		}
	}

	// Build explicit matchup rules for same-division pairs not already covered.
	// This replaces the single default_games_per_pair approach so each division
	// can have its own games_per_pair computed from its own team count.
	type teamPair struct{ a, b string }
	existingPairs := make(map[teamPair]bool)
	for _, mr := range allMatchupRules {
		existingPairs[teamPair{mr.TeamAID, mr.TeamBID}] = true
		existingPairs[teamPair{mr.TeamBID, mr.TeamAID}] = true
	}

	// Group teams by division for per-division computations
	teamsByDiv := make(map[string][]Team)
	for _, t := range allTeams {
		teamsByDiv[t.DivisionID] = append(teamsByDiv[t.DivisionID], t)
	}

	for _, divID := range season.DivisionIDs {
		divTeams := teamsByDiv[divID]
		if len(divTeams) < 2 {
			continue
		}
		totalRequired := 0
		for _, t := range divTeams {
			totalRequired += t.GamesRequired
		}
		avgRequired := totalRequired / len(divTeams)
		nPairs := len(divTeams) - 1
		gamesPerPair := (avgRequired + nPairs/2) / nPairs
		if gamesPerPair < 1 {
			gamesPerPair = 1
		}

		for a := 0; a < len(divTeams); a++ {
			for b := a + 1; b < len(divTeams); b++ {
				p := teamPair{divTeams[a].ID, divTeams[b].ID}
				if !existingPairs[p] {
					allMatchupRules = append(allMatchupRules, MatchupRule{
						TeamAID:  divTeams[a].ID,
						TeamBID:  divTeams[b].ID,
						MinGames: gamesPerPair,
						MaxGames: gamesPerPair,
					})
					existingPairs[p] = true
					existingPairs[teamPair{p.b, p.a}] = true
				}
			}
		}
		log.Printf("division %s: %d teams, games_per_pair=%d", divLabel(divID), len(divTeams), gamesPerPair)
	}

	solverMatchupRules := make([]SolverMatchupRule, len(allMatchupRules))
	for i, r := range allMatchupRules {
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
			solverSlots[j] = SolverFieldSlot(slot)
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

	// Auto-inject prefer_fields soft constraint when any division has preferred fields.
	if len(divisionPreferredFields) > 0 {
		hasPrefFields := false
		for _, c := range solverConstraints {
			if c.Type == "prefer_fields" {
				hasPrefFields = true
				break
			}
		}
		if !hasPrefFields {
			params, _ := json.Marshal(map[string]interface{}{
				"division_preferred_fields": divisionPreferredFields,
			})
			solverConstraints = append(solverConstraints, SolverConstraint{
				Type:   "prefer_fields",
				Params: json.RawMessage(params),
				IsHard: false,
				Weight: 1.0,
			})
			log.Printf("auto-injected prefer_fields constraint for %d divisions", len(divisionPreferredFields))
		}
	}

	// Pre-flight checks per division
	blackoutSet := make(map[string]bool, len(blackoutDates))
	for _, b := range blackoutDates {
		blackoutSet[b] = true
	}
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
	maxPerWeek := 2
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

	for _, divID := range season.DivisionIDs {
		divTeams := teamsByDiv[divID]
		if len(divTeams) < 2 {
			continue
		}
		totalRequired := 0
		for _, t := range divTeams {
			totalRequired += t.GamesRequired
		}
		avgRequired := totalRequired / len(divTeams)
		nPairs := len(divTeams) - 1
		gamesPerPair := (avgRequired + nPairs/2) / nPairs
		if gamesPerPair < 1 {
			gamesPerPair = 1
		}

		maxAchievable := maxPerWeek * seasonWeeks
		if gamesPerPair > maxAchievable {
			return fmt.Errorf(
				"infeasible for division %s: need %d games per team but max_games_per_team_per_week=%d over %d weeks only allows %d",
				divLabel(divID), gamesPerPair, maxPerWeek, seasonWeeks, maxAchievable,
			)
		}

		// Determine which fields this division can use
		divAllowed, hasRestrictions := divisionFieldRestrictions[divID]
		divFieldSet := make(map[string]bool)
		if hasRestrictions {
			for _, fid := range divAllowed {
				divFieldSet[fid] = true
			}
		}

		totalGamesRequired := len(divTeams) * (len(divTeams) - 1) / 2 * gamesPerPair
		totalFieldCapacity := 0
		for _, field := range activeFields {
			if hasRestrictions && !divFieldSet[field.ID] {
				continue // this field is not accessible to this division
			}
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
				"infeasible for division %s: need %d game slots (%d pairs × %d games/pair) but accessible fields only provide %d effective slots",
				divLabel(divID), totalGamesRequired, len(divTeams)*(len(divTeams)-1)/2, gamesPerPair, totalFieldCapacity,
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
		DivisionFieldRestrictions: divisionFieldRestrictions,
		DivisionPreferredFields:   divisionPreferredFields,
	}

	// 7. Call scheduler-engine
	log.Printf("Calling scheduler-engine for season %s (%d teams across %d divisions, %d fields)",
		seasonID, len(solverTeams), len(season.DivisionIDs), len(solverFields))
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

	// Build home-team → division lookup for game records
	teamDivision := make(map[string]string, len(allTeams))
	for _, t := range allTeams {
		teamDivision[t.ID] = t.DivisionID
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
			DivisionID:    teamDivision[sg.HomeTeamID],
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
