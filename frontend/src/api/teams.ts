import { teamsApi } from './client'

export interface Division {
  id: string
  name: string
  season_year: number
  season_id?: string
  created_at: string
}

export interface Team {
  id: string
  division_id: string
  name: string
  short_code: string
  team_type: 'local' | 'interleague'
  home_field_id?: string
}

export interface MatchupRule {
  id: string
  team_a_id: string
  team_b_id: string
  min_games: number
  max_games: number
}

export interface DivisionFieldRule {
  id: string
  division_id: string
  field_id: string
  rule_type: 'allowed' | 'preferred'
  created_at: string
}

export const divisionsApi = {
  list: (params?: { season_id?: string }) => {
    const qs = params?.season_id ? `?season_id=${params.season_id}` : ''
    return teamsApi<Division[]>(`/divisions${qs}`)
  },
  create: (data: Omit<Division, 'id' | 'created_at'>) =>
    teamsApi<Division>('/divisions', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Omit<Division, 'id' | 'created_at'>>) =>
    teamsApi<Division>(`/divisions/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => teamsApi<void>(`/divisions/${id}`, { method: 'DELETE' }),
  getTeamsWithRules: (id: string) =>
    teamsApi<{ teams: Team[]; matchup_rules: MatchupRule[] }>(`/divisions/${id}/teams-with-rules`),
  listFieldRules: (divisionId: string) =>
    teamsApi<DivisionFieldRule[]>(`/divisions/${divisionId}/field-rules`),
  createFieldRule: (divisionId: string, data: { field_id: string; rule_type: 'allowed' | 'preferred' }) =>
    teamsApi<DivisionFieldRule>(`/divisions/${divisionId}/field-rules`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  deleteFieldRule: (divisionId: string, ruleId: string) =>
    teamsApi<void>(`/divisions/${divisionId}/field-rules/${ruleId}`, { method: 'DELETE' }),
}

export const teamsApiClient = {
  list: (params?: { division_id?: string }) => {
    const qs = params?.division_id ? `?division_id=${params.division_id}` : ''
    return teamsApi<Team[]>(`/teams${qs}`)
  },
  create: (data: Omit<Team, 'id'>) =>
    teamsApi<Team>('/teams', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Omit<Team, 'id'>>) =>
    teamsApi<Team>(`/teams/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => teamsApi<void>(`/teams/${id}`, { method: 'DELETE' }),
  createMatchupRule: (teamAId: string, data: { team_b_id: string; min_games: number; max_games: number }) =>
    teamsApi<MatchupRule>(`/teams/${teamAId}/matchup-rules`, { method: 'POST', body: JSON.stringify(data) }),
  deleteMatchupRule: (teamAId: string, ruleId: string) =>
    teamsApi<void>(`/teams/${teamAId}/matchup-rules/${ruleId}`, { method: 'DELETE' }),
}
