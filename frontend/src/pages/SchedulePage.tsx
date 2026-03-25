import { useState, useEffect, Fragment } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  seasonsApi, gamesApi, constraintsApi, seasonBlackoutsApi,
  type Season, type Game, type GamesSummaryResponse, type SeasonConstraint, type SeasonBlackout,
} from '../api/schedule'
import { teamsApiClient, divisionsApi, type Team, type Division } from '../api/teams'
import { fieldsApiClient, type Field } from '../api/fields'

interface EditForm {
  game_date: string
  start_time: string
  field_id: string
  status: string
}

interface AddForm {
  division_id: string
  home_team_id: string
  away_team_id: string
  field_id: string
  start_time: string
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

  // #28 constraints, #30 blackouts
  const [constraints, setConstraints] = useState<SeasonConstraint[]>([])
  const [seasonBlackouts, setSeasonBlackouts] = useState<SeasonBlackout[]>([])
  const [showConstraints, setShowConstraints] = useState(false)

  // #32 sort toggle
  const [sortOrder, setSortOrder] = useState<'time' | 'division'>('time')

  // #29 add game
  const [addingGameDate, setAddingGameDate] = useState<string | null>(null)
  const [addForm, setAddForm] = useState<AddForm>({ division_id: '', home_team_id: '', away_team_id: '', field_id: '', start_time: '' })

  useEffect(() => {
    loadInitial()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (selectedSeasonId) {
      setSearchParams({ season: selectedSeasonId })
      loadGames(selectedSeasonId)
      loadSeasonDetail(selectedSeasonId)
    } else {
      setGames([])
      setConstraints([])
      setSeasonBlackouts([])
    }
    setConflicts(null)
    setSummary(null)
    setEditingGameId(null)
    setAddingGameDate(null)
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

  async function loadSeasonDetail(seasonId: string) {
    try {
      const [c, b] = await Promise.all([
        constraintsApi.list(seasonId),
        seasonBlackoutsApi.list(seasonId),
      ])
      setConstraints(c)
      setSeasonBlackouts(b)
    } catch (e) {
      setError(String(e))
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
        gamesApi.update(selectedSeasonId, gameA.id, { ...gameA, start_time: gameB.start_time, manually_edited: true }),
        gamesApi.update(selectedSeasonId, gameB.id, { ...gameB, start_time: gameA.start_time, manually_edited: true }),
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

  async function createGame() {
    if (!selectedSeasonId || !addingGameDate) return
    if (!addForm.home_team_id || !addForm.away_team_id || !addForm.field_id || !addForm.start_time || !addForm.division_id) return
    try {
      const created = await gamesApi.create(selectedSeasonId, {
        home_team_id: addForm.home_team_id,
        away_team_id: addForm.away_team_id,
        field_id: addForm.field_id,
        game_date: addingGameDate,
        start_time: addForm.start_time,
        division_id: addForm.division_id,
      })
      setGames(prev => [...prev, created])
      setAddingGameDate(null)
      setAddForm({ division_id: '', home_team_id: '', away_team_id: '', field_id: '', start_time: '' })
    } catch (e) {
      setError(String(e))
    }
  }

  // Group games by date
  const gamesByDate = games.reduce<Record<string, Game[]>>((acc, game) => {
    const d = game.game_date
    if (!acc[d]) acc[d] = []
    acc[d].push(game)
    return acc
  }, {})

  // Sort within each date: by division name then time, or just time
  for (const date of Object.keys(gamesByDate)) {
    gamesByDate[date].sort((a, b) => {
      if (sortOrder === 'division') {
        const divA = divisions[a.division_id]?.name ?? ''
        const divB = divisions[b.division_id]?.name ?? ''
        if (divA !== divB) return divA.localeCompare(divB)
      }
      return a.start_time.localeCompare(b.start_time)
    })
  }

  const sortedDates = Object.keys(gamesByDate).sort()

  // Merged date list for time-first view (game days + blackout days)
  const blackoutDateSet = new Set(seasonBlackouts.map(b => b.blackout_date))
  const allDates = Array.from(new Set([...sortedDates, ...Array.from(blackoutDateSet)])).sort()

  // Division order for division-first view
  const selectedSeason = seasons.find(s => s.id === selectedSeasonId)
  const orderedDivisions = (selectedSeason?.division_ids ?? [])
    .map(id => divisions[id])
    .filter((d): d is Division => !!d)
    .sort((a, b) => a.name.localeCompare(b.name))

  // Double-header detection
  const doubleHeaderKeys = new Set<string>()
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

  // Teams in the currently selected division for the add-game form
  const addFormTeams = addForm.division_id
    ? Object.values(teams).filter(t => t.division_id === addForm.division_id)
    : []

  function renderGameTable(dateGames: Game[], date: string) {
    return (
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
          {dateGames.map((game, idx) => (
            <Fragment key={game.id}>
              <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                <td style={tdStyle}>{game.start_time}</td>
                <td style={{ ...tdStyle, color: '#555', fontSize: '0.83rem' }}>
                  {divisions[game.division_id]?.name ?? game.division_id ?? '—'}
                </td>
                <td style={tdStyle}>
                  {doubleHeaderKeys.has(`${game.game_date}:${game.home_team_id}`)
                    ? <span style={{ background: '#fff3cd', borderRadius: 3, padding: '0 3px' }} title="Plays twice today">{teams[game.home_team_id]?.name || game.home_team_id}</span>
                    : (teams[game.home_team_id]?.name || game.home_team_id)
                  }
                </td>
                <td style={tdStyle}>
                  {doubleHeaderKeys.has(`${game.game_date}:${game.away_team_id}`)
                    ? <span style={{ background: '#fff3cd', borderRadius: 3, padding: '0 3px' }} title="Plays twice today">{teams[game.away_team_id]?.name || game.away_team_id}</span>
                    : (teams[game.away_team_id]?.name || game.away_team_id)
                  }
                </td>
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
                    <span style={{ background: '#fff3cd', color: '#856404', padding: '1px 8px', borderRadius: 10, fontSize: '0.75rem', marginLeft: game.is_interleague ? 4 : 0 }} title="Team(s) play twice today">
                      double-header
                    </span>
                  )}
                </td>
                <td style={{ ...tdStyle, whiteSpace: 'nowrap' }}>
                  {editingGameId !== game.id && sortOrder === 'time' && (
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
                        disabled={idx === dateGames.length - 1}
                        className="btn"
                        style={{ fontSize: '0.75rem', padding: '1px 6px', marginRight: 4, opacity: idx === dateGames.length - 1 ? 0.3 : 1 }}
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
    )
  }

  function renderDateCard(date: string, dateGames: Game[], isBlackout: boolean, defaultDivisionId?: string) {
    const isGameDay = dateGames.length > 0
    const formattedDate = new Date(date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })
    const isAddingHere = addingGameDate === date

    if (!isGameDay && isBlackout) {
      return (
        <div key={date} className="card" style={{ padding: '0.75rem 1.5rem', background: '#fff8e1', borderLeft: '4px solid #ffc107' }}>
          <span style={{ fontSize: '0.9rem', color: '#856404', fontWeight: 600 }}>{formattedDate}</span>
          <span style={{ marginLeft: '0.75rem', background: '#ffc107', color: '#333', padding: '1px 8px', borderRadius: 10, fontSize: '0.75rem', fontWeight: 600 }}>blackout</span>
        </div>
      )
    }

    return (
      <div key={date} className="card" style={{ padding: '1rem 1.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <h3 style={{ margin: 0, color: '#1a5276', fontSize: '1rem' }}>
            {formattedDate}
            <span style={{ fontWeight: 400, color: '#888', marginLeft: '0.5rem', fontSize: '0.85rem' }}>
              ({dateGames.length} game{dateGames.length !== 1 ? 's' : ''})
            </span>
            {isBlackout && (
              <span style={{ marginLeft: '0.5rem', background: '#ffc107', color: '#333', padding: '1px 8px', borderRadius: 10, fontSize: '0.75rem' }}>blackout</span>
            )}
          </h3>
          {selectedSeasonId && (
            <button
              onClick={() => {
                if (isAddingHere) {
                  setAddingGameDate(null)
                } else {
                  setAddingGameDate(date)
                  setAddForm({ division_id: defaultDivisionId ?? '', home_team_id: '', away_team_id: '', field_id: '', start_time: '' })
                }
              }}
              className="btn btn-primary"
              style={{ fontSize: '0.78rem', padding: '2px 10px' }}
            >
              {isAddingHere ? 'Cancel' : '+ Add Game'}
            </button>
          )}
        </div>

        {renderGameTable(dateGames, date)}

        {isAddingHere && (
          <div style={{ marginTop: '0.75rem', padding: '0.75rem', background: '#f0f4f8', borderRadius: 6 }}>
            <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
              <label style={{ fontSize: '0.85rem' }}>
                Division<br />
                <select
                  value={addForm.division_id}
                  onChange={e => setAddForm(p => ({ ...p, division_id: e.target.value, home_team_id: '', away_team_id: '' }))}
                  style={inputStyle}
                >
                  <option value="">— Select —</option>
                  {Object.values(divisions).sort((a, b) => a.name.localeCompare(b.name)).map(d => (
                    <option key={d.id} value={d.id}>{d.name}</option>
                  ))}
                </select>
              </label>
              <label style={{ fontSize: '0.85rem' }}>
                Home Team<br />
                <select
                  value={addForm.home_team_id}
                  onChange={e => setAddForm(p => ({ ...p, home_team_id: e.target.value }))}
                  style={inputStyle}
                  disabled={!addForm.division_id}
                >
                  <option value="">— Select —</option>
                  {addFormTeams.map(t => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
              </label>
              <label style={{ fontSize: '0.85rem' }}>
                Away Team<br />
                <select
                  value={addForm.away_team_id}
                  onChange={e => setAddForm(p => ({ ...p, away_team_id: e.target.value }))}
                  style={inputStyle}
                  disabled={!addForm.division_id}
                >
                  <option value="">— Select —</option>
                  {addFormTeams.map(t => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
              </label>
              <label style={{ fontSize: '0.85rem' }}>
                Field<br />
                <select
                  value={addForm.field_id}
                  onChange={e => setAddForm(p => ({ ...p, field_id: e.target.value }))}
                  style={inputStyle}
                >
                  <option value="">— Select —</option>
                  {Object.values(fields).map(f => (
                    <option key={f.id} value={f.id}>{f.name}</option>
                  ))}
                </select>
              </label>
              <label style={{ fontSize: '0.85rem' }}>
                Time<br />
                <input
                  type="time"
                  value={addForm.start_time}
                  onChange={e => setAddForm(p => ({ ...p, start_time: e.target.value }))}
                  style={inputStyle}
                />
              </label>
              <button
                onClick={createGame}
                className="btn btn-primary"
                style={{ fontSize: '0.85rem' }}
                disabled={!addForm.home_team_id || !addForm.away_team_id || !addForm.field_id || !addForm.start_time}
              >
                Add Game
              </button>
            </div>
          </div>
        )}
      </div>
    )
  }

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
              <label>
                <strong>View</strong><br />
                <div style={{ display: 'flex', borderRadius: 4, overflow: 'hidden', border: '1px solid #ccc' }}>
                  <button
                    onClick={() => setSortOrder('time')}
                    style={{ padding: '0.35rem 0.75rem', fontSize: '0.85rem', border: 'none', cursor: 'pointer', background: sortOrder === 'time' ? '#1a5276' : '#fff', color: sortOrder === 'time' ? '#fff' : '#333' }}
                  >By Time</button>
                  <button
                    onClick={() => setSortOrder('division')}
                    style={{ padding: '0.35rem 0.75rem', fontSize: '0.85rem', border: 'none', borderLeft: '1px solid #ccc', cursor: 'pointer', background: sortOrder === 'division' ? '#1a5276' : '#fff', color: sortOrder === 'division' ? '#fff' : '#333' }}
                  >By Division</button>
                </div>
              </label>
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

      {/* Constraints panel (#28) */}
      {selectedSeasonId && constraints.length > 0 && (
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <strong style={{ fontSize: '0.95rem' }}>Constraints ({constraints.length})</strong>
            <button
              onClick={() => setShowConstraints(p => !p)}
              className="btn"
              style={{ fontSize: '0.8rem', padding: '2px 10px', background: '#eee' }}
            >
              {showConstraints ? 'Hide' : 'Show'}
            </button>
          </div>
          {showConstraints && (
            <div style={{ marginTop: '0.5rem', display: 'flex', flexDirection: 'column', gap: '0.3rem' }}>
              {constraints.map(c => {
                const isAuto = c.params.auto_injected === true
                return (
                  <div key={c.id} style={{ fontSize: '0.85rem', opacity: isAuto ? 0.7 : 1 }}>
                    <strong>{c.type.replace(/_/g, ' ')}</strong>
                    {isAuto && <span style={{ marginLeft: '0.4rem', background: '#e9ecef', color: '#555', padding: '1px 6px', borderRadius: 10, fontSize: '0.72rem' }}>auto</span>}
                    <span style={{
                      marginLeft: '0.4rem',
                      background: c.is_hard ? '#fde8d8' : '#e8f4d8',
                      color: c.is_hard ? '#8b3a0a' : '#2e5e0a',
                      padding: '1px 6px', borderRadius: 10, fontSize: '0.72rem',
                    }}>{c.is_hard ? 'hard' : `soft w=${c.weight}`}</span>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}

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

      {/* Empty states */}
      {!selectedSeasonId && (
        <div className="card">
          <p className="placeholder">Select a season above to view its schedule.</p>
        </div>
      )}
      {selectedSeasonId && gamesLoading && (
        <div className="card"><p>Loading games…</p></div>
      )}
      {selectedSeasonId && !gamesLoading && games.length === 0 && seasonBlackouts.length === 0 && (
        <div className="card">
          <p className="placeholder">No games scheduled yet. Generate a schedule from the Seasons page.</p>
        </div>
      )}

      {/* Time-first view: date cards interleaved with blackout cards (#30) */}
      {selectedSeasonId && !gamesLoading && sortOrder === 'time' && allDates.map(date => {
        const dateGames = gamesByDate[date] ?? []
        const isBlackout = blackoutDateSet.has(date)
        return renderDateCard(date, dateGames, isBlackout)
      })}

      {/* Division-first view (#32) */}
      {selectedSeasonId && !gamesLoading && sortOrder === 'division' && (
        <>
          {orderedDivisions.length === 0
            ? allDates.map(date => renderDateCard(date, gamesByDate[date] ?? [], blackoutDateSet.has(date)))
            : orderedDivisions.map(div => {
                const divGames = games.filter(g => g.division_id === div.id)
                if (divGames.length === 0) return null
                const divDates = [...new Set(divGames.map(g => g.game_date))].sort()
                return (
                  <div key={div.id}>
                    <h2 style={{ margin: '1.5rem 0 0.5rem', fontSize: '1.1rem', color: '#1a5276', borderBottom: '2px solid #e0e7ef', paddingBottom: '0.25rem' }}>
                      {div.name}
                    </h2>
                    {divDates.map(date => {
                      const dateGames = divGames.filter(g => g.game_date === date).sort((a, b) => a.start_time.localeCompare(b.start_time))
                      return renderDateCard(date, dateGames, false, div.id)
                    })}
                  </div>
                )
              })
          }
          {seasonBlackouts.length > 0 && (
            <div>
              <h2 style={{ margin: '1.5rem 0 0.5rem', fontSize: '1.1rem', color: '#856404', borderBottom: '2px solid #ffc107', paddingBottom: '0.25rem' }}>
                Blackout Dates
              </h2>
              {[...seasonBlackouts].sort((a, b) => a.blackout_date.localeCompare(b.blackout_date)).map(b => (
                <div key={b.id} className="card" style={{ padding: '0.75rem 1.5rem', background: '#fff8e1', borderLeft: '4px solid #ffc107' }}>
                  <span style={{ fontSize: '0.9rem', color: '#856404' }}>
                    {new Date(b.blackout_date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
                  </span>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
