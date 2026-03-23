const BASE_URLS = {
  teams: '/api/teams',
  fields: '/api/fields',
  schedule: '/api/schedule',
}

async function request<T>(
  service: keyof typeof BASE_URLS,
  path: string,
  options?: RequestInit,
): Promise<T> {
  const url = `${BASE_URLS[service]}${path}`
  const response = await fetch(url, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })
  if (!response.ok) {
    const error = await response.text()
    throw new Error(`${response.status} ${response.statusText}: ${error}`)
  }
  if (response.status === 204) return undefined as T
  return response.json()
}

export function teamsApi<T>(path: string, options?: RequestInit) {
  return request<T>('teams', path, options)
}

export function fieldsApi<T>(path: string, options?: RequestInit) {
  return request<T>('fields', path, options)
}

export function scheduleApi<T>(path: string, options?: RequestInit) {
  return request<T>('schedule', path, options)
}
