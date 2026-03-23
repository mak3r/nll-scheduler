import { teamsApi, fieldsApi, scheduleApi } from './client'

export interface TeamsBundle {
  divisions: unknown[]
  teams: unknown[]
  matchup_rules: unknown[]
}

export interface FieldsBundle {
  fields: unknown[]
  availability_windows: unknown[]
  blackout_dates: unknown[]
}

export interface ScheduleBundle {
  seasons: unknown[]
}

export interface FullBundle {
  version: '1'
  exported_at: string
  teams: TeamsBundle
  fields: FieldsBundle
  schedule: ScheduleBundle
}

export const adminApi = {
  exportAll: async (): Promise<FullBundle> => {
    const [teams, fields, schedule] = await Promise.all([
      teamsApi<TeamsBundle>('/export'),
      fieldsApi<FieldsBundle>('/export'),
      scheduleApi<ScheduleBundle>('/export'),
    ])
    return {
      version: '1',
      exported_at: new Date().toISOString(),
      teams,
      fields,
      schedule,
    }
  },

  importAll: async (bundle: FullBundle): Promise<void> => {
    // Sequential: teams → fields → schedule (schedule games reference team/field IDs)
    await teamsApi('/import', {
      method: 'POST',
      body: JSON.stringify(bundle.teams),
    })
    await fieldsApi('/import', {
      method: 'POST',
      body: JSON.stringify(bundle.fields),
    })
    await scheduleApi('/import', {
      method: 'POST',
      body: JSON.stringify(bundle.schedule),
    })
  },
}
