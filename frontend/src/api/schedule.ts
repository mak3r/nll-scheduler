import { scheduleApi } from './client'

export interface Season {
  id: string
  name: string
  division_ids: string[]
  start_date: string
  end_date: string
  status: 'draft' | 'generating' | 'review' | 'published'
  is_current: boolean
  created_at: string
}

export interface SeasonConstraint {
  id: string
  season_id: string
  type: string
  params: Record<string, unknown>
  is_hard: boolean
  weight: number
}

export interface Game {
  id: string
  season_id: string
  home_team_id: string
  away_team_id: string
  field_id: string
  game_date: string
  start_time: string
  status: 'scheduled' | 'cancelled' | 'completed'
  division_id: string
  is_interleague: boolean
  manually_edited: boolean
}

export interface TeamSummaryEntry {
  team_id: string
  home: number
  away: number
  total: number
}

export interface DivisionSummary {
  division_id: string
  teams: TeamSummaryEntry[]
}

export interface GamesSummaryResponse {
  divisions: DivisionSummary[]
}

export interface GenerationRun {
  id: string
  season_id: string
  status: 'pending' | 'running' | 'success' | 'failed'
  solver_stats?: Record<string, unknown>
  error_message?: string
}

export const seasonsApi = {
  list: () => scheduleApi<Season[]>('/seasons'),
  create: (data: Omit<Season, 'id' | 'status' | 'is_current' | 'created_at'>) =>
    scheduleApi<Season>('/seasons', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Omit<Season, 'id' | 'created_at'>>) =>
    scheduleApi<Season>(`/seasons/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => scheduleApi<void>(`/seasons/${id}`, { method: 'DELETE' }),
  setCurrent: (id: string) => scheduleApi<Season>(`/seasons/${id}/set-current`, { method: 'POST' }),
}

export const constraintsApi = {
  list: (seasonId: string) =>
    scheduleApi<SeasonConstraint[]>(`/seasons/${seasonId}/constraints`),
  create: (seasonId: string, data: Omit<SeasonConstraint, 'id' | 'season_id'>) =>
    scheduleApi<SeasonConstraint>(`/seasons/${seasonId}/constraints`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  delete: (seasonId: string, constraintId: string) =>
    scheduleApi<void>(`/seasons/${seasonId}/constraints/${constraintId}`, { method: 'DELETE' }),
}

export const gamesApi = {
  list: (seasonId: string) => scheduleApi<Game[]>(`/seasons/${seasonId}/games`),
  update: (seasonId: string, gameId: string, data: Partial<Omit<Game, 'id' | 'season_id'>>) =>
    scheduleApi<Game>(`/seasons/${seasonId}/games/${gameId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  create: (seasonId: string, data: { home_team_id: string; away_team_id: string; field_id: string; game_date: string; start_time: string; division_id: string }) =>
    scheduleApi<Game>(`/seasons/${seasonId}/games`, { method: 'POST', body: JSON.stringify(data) }),
  delete: (seasonId: string, gameId: string) =>
    scheduleApi<void>(`/seasons/${seasonId}/games/${gameId}`, { method: 'DELETE' }),
  checkConflicts: (seasonId: string) =>
    scheduleApi<{ conflicts: string[] }>(`/seasons/${seasonId}/games/check-conflicts`, {
      method: 'POST',
    }),
  summary: (seasonId: string) =>
    scheduleApi<GamesSummaryResponse>(`/seasons/${seasonId}/games/summary`),
  export: (seasonId: string, format: 'json' | 'csv') =>
    scheduleApi<unknown>(`/seasons/${seasonId}/export?format=${format}`),
}

export interface SeasonBlackout {
  id: string
  season_id: string
  blackout_date: string
  created_at: string
}

export interface PreferredDate {
  id: string
  season_id: string
  preferred_date: string
  weight: number
  created_at: string
}

export const seasonBlackoutsApi = {
  list: (seasonId: string) =>
    scheduleApi<SeasonBlackout[]>(`/seasons/${seasonId}/blackout-dates`),
  create: (seasonId: string, blackout_date: string) =>
    scheduleApi<SeasonBlackout>(`/seasons/${seasonId}/blackout-dates`, {
      method: 'POST',
      body: JSON.stringify({ blackout_date }),
    }),
  delete: (seasonId: string, blackoutId: string) =>
    scheduleApi<void>(`/seasons/${seasonId}/blackout-dates/${blackoutId}`, { method: 'DELETE' }),
}

export const preferredDatesApi = {
  list: (seasonId: string) =>
    scheduleApi<PreferredDate[]>(`/seasons/${seasonId}/preferred-interleague-dates`),
  create: (seasonId: string, preferred_date: string, weight = 1.0) =>
    scheduleApi<PreferredDate>(`/seasons/${seasonId}/preferred-interleague-dates`, {
      method: 'POST',
      body: JSON.stringify({ preferred_date, weight }),
    }),
  delete: (seasonId: string, prefId: string) =>
    scheduleApi<void>(`/seasons/${seasonId}/preferred-interleague-dates/${prefId}`, {
      method: 'DELETE',
    }),
}

export const generationApi = {
  start: (seasonId: string) =>
    scheduleApi<{ run_id: string }>(`/seasons/${seasonId}/generate`, { method: 'POST' }),
  getStatus: (seasonId: string, runId: string) =>
    scheduleApi<GenerationRun>(`/seasons/${seasonId}/generate/${runId}`),
}

export interface DivisionGamesRequired {
  id: string
  season_id: string
  division_id: string
  games_required: number
}

export const divisionGamesRequiredApi = {
  list: (seasonId: string) =>
    scheduleApi<DivisionGamesRequired[]>(`/seasons/${seasonId}/division-games-required`),
  upsert: (seasonId: string, divisionId: string, gamesRequired: number) =>
    scheduleApi<DivisionGamesRequired>(`/seasons/${seasonId}/division-games-required/${divisionId}`, {
      method: 'PUT',
      body: JSON.stringify({ games_required: gamesRequired }),
    }),
}
