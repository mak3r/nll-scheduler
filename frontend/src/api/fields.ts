import { fieldsApi } from './client'

export interface Field {
  id: string
  name: string
  address?: string
  max_games_per_day: number
  is_active: boolean
}

export interface AvailabilityWindow {
  id: string
  field_id: string
  window_type: 'recurring' | 'oneoff'
  days_of_week: number[]
  start_date: string
  end_date: string
  start_time: string
  end_time: string
}

export interface BlackoutDate {
  id: string
  field_id: string
  blackout_date: string
  reason?: string
}

export interface AvailableSlot {
  field_id: string
  date: string
  start_time: string
  end_time: string
}

export const fieldsApiClient = {
  list: () => fieldsApi<Field[]>('/fields'),
  create: (data: Omit<Field, 'id'>) =>
    fieldsApi<Field>('/fields', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Omit<Field, 'id'>>) =>
    fieldsApi<Field>(`/fields/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => fieldsApi<void>(`/fields/${id}`, { method: 'DELETE' }),
  getAvailableDatesBulk: (params: { start: string; end: string; field_ids: string[] }) => {
    const qs = `?start=${params.start}&end=${params.end}&field_ids=${params.field_ids.join(',')}`
    return fieldsApi<Record<string, AvailableSlot[]>>(`/fields/available-dates-bulk${qs}`)
  },
}

export const availabilityApi = {
  list: (fieldId: string) =>
    fieldsApi<AvailabilityWindow[]>(`/fields/${fieldId}/availability-windows`),
  create: (fieldId: string, data: Omit<AvailabilityWindow, 'id' | 'field_id'>) =>
    fieldsApi<AvailabilityWindow>(`/fields/${fieldId}/availability-windows`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  delete: (fieldId: string, windowId: string) =>
    fieldsApi<void>(`/fields/${fieldId}/availability-windows/${windowId}`, { method: 'DELETE' }),
}

export const blackoutApi = {
  list: (fieldId: string) => fieldsApi<BlackoutDate[]>(`/fields/${fieldId}/blackout-dates`),
  create: (fieldId: string, data: { blackout_date: string; reason?: string }) =>
    fieldsApi<BlackoutDate>(`/fields/${fieldId}/blackout-dates`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  delete: (fieldId: string, blackoutId: string) =>
    fieldsApi<void>(`/fields/${fieldId}/blackout-dates/${blackoutId}`, { method: 'DELETE' }),
}
