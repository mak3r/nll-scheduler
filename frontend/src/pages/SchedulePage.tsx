import { useState, useEffect, Fragment } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  seasonsApi, gamesApi,
  type Season, type Game, type GamesSummaryResponse,
} from '../api/schedule'
import { teamsApiClient, divisionsApi, type Team, type Division } from '../api/teams'
import { fieldsApiClient, type Field } from '../api/fields'

interface EditForm {
  game_date: string
  start_time: string
  field_id: string
  status: string
}

export default function SchedulePage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const [seasons, setSeasons] = useState<Season[]>([])
  const [selectedSeasonId, setSelectedSeasonId] = useState<string>(searchParams.get('season') || '')
  const [games, setGames] = useState<Game[]>([])
  const [teams, setTeams] = useState<Record<string, Team>>({})
  const [fields, setFields] = useState<Record<string, Field>>({})
  const [divisions, setDivisions] = useState<Record<string, Division>>({})

  const [loading, setLoading] = useState(true)
  const [gamesLoading, setGamesLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [editingGameId, setEditingGameId] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<EditForm>({ game_date: '', start_time: '', field_id: '', status: 'scheduled' })

  const [conflicts, setConflicts] = useState<string[] | null>(null)
  const [conflictsLoading, setConflictsLoading] = useState(false)

  const [summary, setSummary] = useState<GamesSummaryResponse | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)

  useEffect(() => {
    loadInitial()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (selectedSeasonId) {
      setSearchParams({ season: selectedSeasonId })
      loadGames(selectedSeasonId)
    } else {
      setGames([])
    }
    setConflicts(null)
    setSummary(null)
    setEditingGameId(null)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedSeasonId])

  async function loadInitial() {
    setLoading(true)
    try {
      const [s, t, f, d] = await Promise.all([
        seasonsApi.list(),
        teamsApiClient.list(),
        fieldsApiClient.list(),
        divisionsApi.list(),
      ])
      setSeasons(s)

      const teamsMap: Record<string, Team> = {}
      for (const team of t) teamsMap[team.id] = team
      setTeams(teamsMap)

      const fieldsMap: Record<string, Field> = {}
      for (const field of f) fieldsMap[field.id] = field
      setFields(fieldsMap)

      const divisionsMap: Record<string, Division> = {}
      for (const div of d) divisionsMap[div.id] = div
      setDivisions(divisionsMap)

      const urlSeason = searchParams.get('season')
      if (urlSeason && s.some(season => season.id === urlSeason)) {
        setSelectedSeasonId(urlSeason)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function loadGames(seasonId: string) {
    setGamesLoading(true)
    try {
      const g = await gamesApi.list(seasonId)
      setGames(g)
    } catch (e) {
      setError(String(e))
    } finally {
      setGamesLoading(false)
    }
  }

  async function deleteGame(gameId: string) {
    if (!confirm('Delete this game?')) return
    if (!selectedSeasonId) return
    try {
      await gamesApi.delete(selectedSeasonId, gameId)
      setGames(prev => prev.filter(g => g.id !== gameId))
    } catch (e) {
      setError(String(e))
    }
  }

  function startEdit(game: Game) {
    setEditingGameId(game.id)
    setEditForm({
      game_date: game.game_date,
      start_time: game.start_time,
      field_id: game.field_id,
      status: game.status,
    })
  }

  function cancelEdit() {
    setEditingGameId(null)
  }

  async function saveEdit(gameId: string) {
    if (!selectedSeasonId) return
    try {
      const updated = await gamesApi.update(selectedSeasonId, gameId, {
        game_date: editForm.game_date,
        start_time: editForm.start_time,
        field_id: editForm.field_id,
        status: editForm.status as Game['status'],
        manually_edited: true,
      })
      setGames(prev => prev.map(g => g.id === gameId ? updated : g))
      setEditingGameId(null)
      setConflicts(null)
    } catch (e) {
      setError(String(e))
    }
  }

  async function moveGame(date: string, idx: number, direction: 'up' | 'down') {
    if (!selectedSeasonId) return
    const dayGames = gamesByDate[date]
    const otherIdx = direction === 'up' ? idx - 1 : idx + 1
    if (otherIdx < 0 || otherIdx >= dayGames.length) return
    const gameA = dayGames[idx]
    const gameB = dayGames[otherIdx]
    try {
      await Promise.all([
        gamesApi.update(selectedSeasonId, gameA.id, { start_time: gameB.start_time, manually_edited: true }),
        gamesApi.update(selectedSeasonId, gameB.id, { start_time: gameA.start_time, manually_edited: true }),
      ])
      await loadGames(selectedSeasonId)
    } catch (e) {
      setError(String(e))
    }
  }

  async function checkConflicts() {
    if (!selectedSeasonId) return
    setConflictsLoading(true)
    setConflicts(null)
    try {
      const result = await gamesApi.checkConflicts(selectedSeasonId)
      setConflicts(result.conflicts)
    } catch (e) {
      setError(String(e))
    } finally {
      setConflictsLoading(false)
    }
  }

  async function showSummary() {
    if (!selectedSeasonId) return
    setSummaryLoading(true)
    setSummary(null)
    try {
      const result = await gamesApi.summary(selectedSeasonId)
      setSummary(result)
    } catch (e) {
      setError(String(e))
    } finally {
      setSummaryLoading(false)
    }
  }

  async function exportJSON() {
    if (!selectedSeasonId) return
    try {
      const data = await gamesApi.export(selectedSeasonId, 'json')
      const json = JSON.stringify(data, null, 2)
      const blob = new Blob([json], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      const season = seasons.find(s => s.id === selectedSeasonId)
      a.href = url
      a.download = `schedule-${season?.name ?? selectedSeasonId}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setError(String(e))
    }
  }

  function exportCSV() {
    if (!selectedSeasonId) return
    const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
    const headers = ['Date', 'Day', 'Start Time', 'Field', 'Division', 'Home Team', 'Away Team', 'Status']
    const rows = games.map(g => {
      const d = new Date(g.game_date + 'T00:00:00')
      return [
        g.game_date,
        DAYS[d.getDay()],
        g.start_time,
        fields[g.field_id]?.name ?? g.field_id,
        divisions[g.division_id]?.name ?? g.division_id,
        teams[g.home_team_id]?.name ?? g.home_team_id,
        teams[g.away_team_id]?.name ?? g.away_team_id,
        g.status,
      ].map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(',')
    })
    const csv = [headers.join(','), ...rows].join('\n')
    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    const season = seasons.find(s => s.id === selectedSeasonId)
    a.href = url
    a.download = `schedule-${season?.name ?? selectedSeasonId}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  // Group games by date
  const gamesByDate = games.reduce<Record<string, Game[]>>((acc, game) => {
    const d = game.game_date
    if (!acc[d]) acc[d] = []
    acc[d].push(game)
    return acc
  }, {})

  const sortedDates = Object.keys(gamesByDate).sort()

  // Detect teams playing more than once on the same day (double-headers)
  const doubleHeaderKeys = new Set<string>() // `${date}:${teamId}`
  for (const date of sortedDates) {
    const counts: Record<string, number> = {}
    for (const g of gamesByDate[date]) {
      if (g.status === 'cancelled') continue
      counts[g.home_team_id] = (counts[g.home_team_id] || 0) + 1
      counts[g.away_team_id] = (counts[g.away_team_id] || 0) + 1
    }
    for (const [teamId, n] of Object.entries(counts)) {
      if (n > 1) doubleHeaderKeys.add(`${date}:${teamId}`)
    }
  }
  function isDoubleHeader(game: Game) {
    return doubleHeaderKeys.has(`${game.game_date}:${game.home_team_id}`) ||
           doubleHeaderKeys.has(`${game.game_date}:${game.away_team_id}`)
  }

  const inputStyle: React.CSSProperties = { padding: '0.35rem', borderRadius: 4, border: '1px solid #ccc', fontSize: '0.85rem' }
  const tdStyle: React.CSSProperties = { padding: '0.4rem 0.5rem', verticalAlign: 'middle' }

  if (loading) return <div className="card"><p>Loading...</p></div>

  const selectedSeason = seasons.find(s => s.id === selectedSeasonId)

  return (
    <div>
      <h1>Schedule</h1>

      {error && (
        <div className="card" style={{ background: '#fdd' }}>
          <strong>Error:</strong> {error}
          <button onClick={() => setError(null)} style={{ marginLeft: '1rem', cursor: 'pointer', background: 'none', border: 'none', fontWeight: 'bold' }}>✕</button>
        </div>
      )}

      {/* Season selector + actions */}
      <div className="card">
        <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <label>
            <strong>Season</strong><br />
            <select
              value={selectedSeasonId}
              onChange={e => setSelectedSeasonId(e.target.value)}
              style={{ ...inputStyle, minWidth: 220, fontSize: '0.95rem', padding: '0.45rem' }}
            >
              <option value="">— Select a season —</option>
              {seasons.map(s => (
                <option key={s.id} value={s.id}>{s.name} ({s.status})</option>
              ))}
            </select>
          </label>

          {selectedSeasonId && (
            <>
              <button onClick={checkConflicts} className="btn btn-primary" disabled={conflictsLoading} style={{ fontSize: '0.85rem' }}>
                {conflictsLoading ? 'Checking…' : 'Check Conflicts'}
              </button>
              <button onClick={showSummary} className="btn btn-primary" disabled={summaryLoading} style={{ fontSize: '0.85rem' }}>
                {summaryLoading ? 'Loading…' : 'Show Summary'}
              </button>
              <button onClick={exportJSON} className="btn btn-primary" style={{ fontSize: '0.85rem' }}>
                Export JSON
              </button>
              <button onClick={exportCSV} className="btn btn-primary" style={{ fontSize: '0.85rem' }}>
                Export CSV
              </button>
            </>
          )}
        </div>

        {selectedSeason && (
          <div style={{ marginTop: '0.5rem', fontSize: '0.85rem', color: '#666' }}>
            {selectedSeason.start_date} &rarr; {selectedSeason.end_date}
            {' '}&middot;{' '}
            <span style={{ fontWeight: 600 }}>{games.length}</span> game{games.length !== 1 ? 's' : ''}
          </div>
        )}
      </div>

      {/* Conflicts panel */}
      {conflicts !== null && (
        <div className="card" style={{ background: conflicts.length === 0 ? '#d4edda' : '#fff3cd' }}>
          {conflicts.length === 0 ? (
            <strong style={{ color: '#155724' }}>No conflicts found.</strong>
          ) : (
            <>
              <strong style={{ color: '#856404' }}>{conflicts.length} conflict{conflicts.length !== 1 ? 's' : ''} found:</strong>
              <ul style={{ margin: '0.5rem 0 0 1.25rem', fontSize: '0.9rem' }}>
                {conflicts.map((c, i) => <li key={i}>{c}</li>)}
              </ul>
            </>
          )}
        </div>
      )}

      {/* Summary panel */}
      {summary !== null && (
        <div className="card">
          <h3 style={{ margin: '0 0 0.75rem', fontSize: '1rem', color: '#1a5276' }}>Home / Away Summary</h3>
          {summary.divisions.map(div => (
            <div key={div.division_id} style={{ marginBottom: '1rem' }}>
              <strong style={{ fontSize: '0.9rem' }}>{divisions[div.division_id]?.name ?? div.division_id}</strong>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem', marginTop: '0.35rem' }}>
                <thead>
                  <tr style={{ background: '#f0f4f8', textAlign: 'left' }}>
                    <th style={tdStyle}>Team</th>
                    <th style={{ ...tdStyle, textAlign: 'center' }}>Home</th>
                    <th style={{ ...tdStyle, textAlign: 'center' }}>Away</th>
                    <th style={{ ...tdStyle, textAlign: 'center' }}>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {[...div.teams].sort((a, b) => b.total - a.total).map(t => (
                    <tr key={t.team_id} style={{ borderBottom: '1px solid #f0f0f0' }}>
                      <td style={tdStyle}>{teams[t.team_id]?.name ?? t.team_id}</td>
                      <td style={{ ...tdStyle, textAlign: 'center' }}>{t.home}</td>
                      <td style={{ ...tdStyle, textAlign: 'center' }}>{t.away}</td>
                      <td style={{ ...tdStyle, textAlign: 'center', fontWeight: 600 }}>{t.total}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </div>
      )}

      {/* Games table */}
      {!selectedSeasonId && (
        <div className="card">
          <p className="placeholder">Select a season above to view its schedule.</p>
        </div>
      )}

      {selectedSeasonId && gamesLoading && (
        <div className="card"><p>Loading games…</p></div>
      )}

      {selectedSeasonId && !gamesLoading && games.length === 0 && (
        <div className="card">
          <p className="placeholder">No games scheduled yet. Generate a schedule from the Seasons page.</p>
        </div>
      )}

      {selectedSeasonId && !gamesLoading && sortedDates.map(date => (
        <div key={date} className="card" style={{ padding: '1rem 1.5rem' }}>
          <h3 style={{ margin: '0 0 0.5rem', color: '#1a5276', fontSize: '1rem' }}>
            {new Date(date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
            <span style={{ fontWeight: 400, color: '#888', marginLeft: '0.5rem', fontSize: '0.85rem' }}>
              ({gamesByDate[date].length} game{gamesByDate[date].length !== 1 ? 's' : ''})
            </span>
          </h3>

          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.88rem' }}>
            <thead>
              <tr style={{ borderBottom: '2px solid #eee', textAlign: 'left', color: '#555' }}>
                <th style={tdStyle}>Time</th>
                <th style={tdStyle}>Division</th>
                <th style={tdStyle}>Home</th>
                <th style={tdStyle}>Away</th>
                <th style={tdStyle}>Field</th>
                <th style={tdStyle}>Status</th>
                <th style={tdStyle}>Type</th>
                <th style={tdStyle}></th>
              </tr>
            </thead>
            <tbody>
              {gamesByDate[date].map((game, idx) => (
                <Fragment key={game.id}>
                  <tr style={{ borderBottom: '1px solid #f0f0f0', background: isDoubleHeader(game) ? '#fffbea' : undefined }}>
                    <td style={tdStyle}>{game.start_time}</td>
                    <td style={{ ...tdStyle, color: '#555', fontSize: '0.83rem' }}>
                      {divisions[game.division_id]?.name ?? game.division_id ?? '—'}
                    </td>
                    <td style={tdStyle}>{teams[game.home_team_id]?.name || game.home_team_id}</td>
                    <td style={tdStyle}>{teams[game.away_team_id]?.name || game.away_team_id}</td>
                    <td style={tdStyle}>{fields[game.field_id]?.name || game.field_id}</td>
                    <td style={tdStyle}>
                      <span style={{
                        background:
                          game.status === 'scheduled' ? '#e8f4f8'
                          : game.status === 'completed' ? '#d4edda'
                          : '#f8d7da',
                        color:
                          game.status === 'scheduled' ? '#1a5276'
                          : game.status === 'completed' ? '#155724'
                          : '#721c24',
                        padding: '1px 8px',
                        borderRadius: 10,
                        fontSize: '0.78rem',
                        fontWeight: 600,
                      }}>
                        {game.status}
                      </span>
                      {game.manually_edited && (
                        <span style={{ marginLeft: 4, fontSize: '0.73rem', color: '#888' }} title="Manually edited">✎</span>
                      )}
                    </td>
                    <td style={tdStyle}>
                      {game.is_interleague && (
                        <span style={{ background: '#e8f4f8', color: '#1a5276', padding: '1px 8px', borderRadius: 10, fontSize: '0.75rem' }}>
                          interleague
                        </span>
                      )}
                      {isDoubleHeader(game) && (
                        <span style={{ background: '#fff3cd', color: '#856404', padding: '1px 8px', borderRadius: 10, fontSize: '0.75rem', marginLeft: game.is_interleague ? 4 : 0 }} title="One or more teams play twice today">
                          double-header
                        </span>
                      )}
                    </td>
                    <td style={{ ...tdStyle, whiteSpace: 'nowrap' }}>
                      {editingGameId !== game.id && (
                        <>
                          <button
                            onClick={() => moveGame(date, idx, 'up')}
                            disabled={idx === 0}
                            className="btn"
                            style={{ fontSize: '0.75rem', padding: '1px 6px', marginRight: 2, opacity: idx === 0 ? 0.3 : 1 }}
                            title="Move earlier"
                          >↑</button>
                          <button
                            onClick={() => moveGame(date, idx, 'down')}
                            disabled={idx === gamesByDate[date].length - 1}
                            className="btn"
                            style={{ fontSize: '0.75rem', padding: '1px 6px', marginRight: 4, opacity: idx === gamesByDate[date].length - 1 ? 0.3 : 1 }}
                            title="Move later"
                          >↓</button>
                        </>
                      )}
                      <button
                        onClick={() => editingGameId === game.id ? cancelEdit() : startEdit(game)}
                        className="btn btn-primary"
                        style={{ fontSize: '0.78rem', padding: '2px 10px', marginRight: 4 }}
                      >
                        {editingGameId === game.id ? 'Cancel' : 'Edit'}
                      </button>
                      <button
                        onClick={() => deleteGame(game.id)}
                        className="btn btn-danger"
                        style={{ fontSize: '0.78rem', padding: '2px 10px' }}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>

                  {editingGameId === game.id && (
                    <tr style={{ background: '#f0f4f8' }}>
                      <td colSpan={8} style={{ padding: '0.75rem 0.5rem' }}>
                        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
                          <label style={{ fontSize: '0.85rem' }}>
                            Date<br />
                            <input
                              type="date"
                              value={editForm.game_date}
                              onChange={e => setEditForm(p => ({ ...p, game_date: e.target.value }))}
                              style={inputStyle}
                            />
                          </label>
                          <label style={{ fontSize: '0.85rem' }}>
                            Time<br />
                            <input
                              type="time"
                              value={editForm.start_time}
                              onChange={e => setEditForm(p => ({ ...p, start_time: e.target.value }))}
                              style={inputStyle}
                            />
                          </label>
                          <label style={{ fontSize: '0.85rem' }}>
                            Field<br />
                            <select
                              value={editForm.field_id}
                              onChange={e => setEditForm(p => ({ ...p, field_id: e.target.value }))}
                              style={inputStyle}
                            >
                              {Object.values(fields).map(f => (
                                <option key={f.id} value={f.id}>{f.name}</option>
                              ))}
                            </select>
                          </label>
                          <label style={{ fontSize: '0.85rem' }}>
                            Status<br />
                            <select
                              value={editForm.status}
                              onChange={e => setEditForm(p => ({ ...p, status: e.target.value }))}
                              style={inputStyle}
                            >
                              <option value="scheduled">Scheduled</option>
                              <option value="cancelled">Cancelled</option>
                              <option value="completed">Completed</option>
                            </select>
                          </label>
                          <button onClick={() => saveEdit(game.id)} className="btn btn-primary" style={{ fontSize: '0.85rem' }}>
                            Save
                          </button>
                          <button onClick={cancelEdit} className="btn" style={{ fontSize: '0.85rem', background: '#eee' }}>
                            Cancel
                          </button>
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  )
}
