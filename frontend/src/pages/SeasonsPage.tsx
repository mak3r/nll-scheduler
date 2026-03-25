import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import {
  seasonsApi,
  constraintsApi,
  generationApi,
  seasonBlackoutsApi,
  preferredDatesApi,
  divisionGamesRequiredApi,
  type Season,
  type SeasonConstraint,
  type SeasonBlackout,
  type PreferredDate,
  type GenerationRun,
  type DivisionGamesRequired,
} from '../api/schedule'
import { divisionsApi, type Division } from '../api/teams'

// Constraint type definitions
interface ConstraintDef {
  label: string
  isHard: boolean
  defaultWeight: number
  defaultParams: Record<string, unknown>
  paramFields: { key: string; label: string; type: 'number' | 'text' }[]
}

const CONSTRAINT_TYPES: Record<string, ConstraintDef> = {
  round_robin_matchup: {
    label: 'Round Robin Matchup',
    isHard: true,
    defaultWeight: 1,
    defaultParams: { default_games_per_pair: 2 },
    paramFields: [{ key: 'default_games_per_pair', label: 'Games per Pair', type: 'number' }],
  },
  max_games_per_field_per_day: {
    label: 'Max Games per Field per Day',
    isHard: true,
    defaultWeight: 1,
    defaultParams: { max_games_per_day: 4 },
    paramFields: [{ key: 'max_games_per_day', label: 'Max Games/Day', type: 'number' }],
  },
  max_games_per_team_per_week: {
    label: 'Max Games per Team per Week (optional cap)',
    isHard: true,
    defaultWeight: 1,
    defaultParams: { max_games_per_week: 3 },
    paramFields: [{ key: 'max_games_per_week', label: 'Max Games/Week', type: 'number' }],
  },
  min_rest_days_between_games: {
    label: 'Min Rest Days Between Games',
    isHard: true,
    defaultWeight: 1,
    defaultParams: { min_rest_days: 1 },
    paramFields: [{ key: 'min_rest_days', label: 'Min Rest Days', type: 'number' }],
  },
  prefer_interleague_dates: {
    label: 'Prefer Interleague Dates (soft)',
    isHard: false,
    defaultWeight: 1,
    defaultParams: { bonus_per_game: 10 },
    paramFields: [{ key: 'bonus_per_game', label: 'Bonus per Game', type: 'number' }],
  },
  even_home_away_balance: {
    label: 'Even Home/Away Balance (soft)',
    isHard: false,
    defaultWeight: 1,
    defaultParams: { penalty_weight: 5 },
    paramFields: [{ key: 'penalty_weight', label: 'Penalty Weight', type: 'number' }],
  },
}

const STATUS_COLORS: Record<string, { bg: string; color: string }> = {
  draft:      { bg: '#eee',    color: '#555' },
  generating: { bg: '#fff3cd', color: '#856404' },
  review:     { bg: '#d1ecf1', color: '#0c5460' },
  published:  { bg: '#d4edda', color: '#155724' },
  failed:     { bg: '#f8d7da', color: '#721c24' },
}

export default function SeasonsPage() {
  const [seasons, setSeasons] = useState<Season[]>([])
  const [divisions, setDivisions] = useState<Division[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Detail data for selected season
  const [blackouts, setBlackouts] = useState<SeasonBlackout[]>([])
  const [preferred, setPreferred] = useState<PreferredDate[]>([])
  const [constraints, setConstraints] = useState<SeasonConstraint[]>([])
  const [detailLoading, setDetailLoading] = useState(false)

  // Create season form
  const [newSeason, setNewSeason] = useState({
    name: '',
    division_ids: [] as string[],
    start_date: '',
    end_date: '',
  })

  // Edit season form
  const [editingSeason, setEditingSeason] = useState(false)
  const [editSeasonForm, setEditSeasonForm] = useState({ name: '', division_ids: [] as string[], start_date: '', end_date: '' })

  // Blackout form
  const [newBlackout, setNewBlackout] = useState('')

  // Preferred date form
  const [newPreferred, setNewPreferred] = useState({ date: '', weight: 1.0 })

  // Constraint form
  const [newConstraintType, setNewConstraintType] = useState('round_robin_matchup')
  const [newConstraintParams, setNewConstraintParams] = useState<Record<string, unknown>>({})

  // Division games required
  const [divGamesReq, setDivGamesReq] = useState<DivisionGamesRequired[]>([])
  const [editingDivGamesReq, setEditingDivGamesReq] = useState<Record<string, number>>({})

  // Generation state
  const [genStatus, setGenStatus] = useState<GenerationRun | null>(null)
  const [genRunId, setGenRunId] = useState<string | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    loadInitial()
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
  }, [])

  useEffect(() => {
    if (selectedId) loadDetail(selectedId)
    else {
      setBlackouts([])
      setPreferred([])
      setConstraints([])
      setDivGamesReq([])
      setEditingDivGamesReq({})
      setGenStatus(null)
      setGenRunId(null)
    }
  }, [selectedId])

  async function loadInitial() {
    setLoading(true)
    try {
      const [s, d] = await Promise.all([seasonsApi.list(), divisionsApi.list()])
      setSeasons(s)
      setDivisions(d)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function loadDetail(sid: string) {
    setDetailLoading(true)
    try {
      const [b, p, c, dgr] = await Promise.all([
        seasonBlackoutsApi.list(sid),
        preferredDatesApi.list(sid),
        constraintsApi.list(sid),
        divisionGamesRequiredApi.list(sid),
      ])
      setBlackouts(b)
      setPreferred(p)
      setConstraints(c)
      setDivGamesReq(dgr)
    } catch (e) {
      setError(String(e))
    } finally {
      setDetailLoading(false)
    }
  }

  async function saveDivGamesReq(divisionId: string) {
    if (!selectedId) return
    const val = editingDivGamesReq[divisionId]
    if (val === undefined || val < 1) return
    try {
      await divisionGamesRequiredApi.upsert(selectedId, divisionId, val)
      const dgr = await divisionGamesRequiredApi.list(selectedId)
      setDivGamesReq(dgr)
      setEditingDivGamesReq(prev => { const n = { ...prev }; delete n[divisionId]; return n })
    } catch (e) {
      setError(String(e))
    }
  }

  async function createSeason(e: React.FormEvent) {
    e.preventDefault()
    try {
      const s = await seasonsApi.create(newSeason)
      setNewSeason({ name: '', division_ids: [], start_date: '', end_date: '' })
      await loadInitial()
      setSelectedId(s.id)
    } catch (e) {
      setError(String(e))
    }
  }

  async function deleteSeason(id: string) {
    if (!confirm('Delete this season and all its games/constraints?')) return
    try {
      await seasonsApi.delete(id)
      if (selectedId === id) setSelectedId(null)
      await loadInitial()
    } catch (e) {
      setError(String(e))
    }
  }

  async function setCurrentSeason(id: string) {
    try {
      await seasonsApi.setCurrent(id)
      await loadInitial()
    } catch (e) {
      setError(String(e))
    }
  }

  function startEditSeason() {
    if (!selectedSeason) return
    setEditSeasonForm({
      name: selectedSeason.name,
      division_ids: selectedSeason.division_ids || [],
      start_date: selectedSeason.start_date,
      end_date: selectedSeason.end_date,
    })
    setEditingSeason(true)
  }

  async function saveEditSeason(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) return
    try {
      await seasonsApi.update(selectedId, editSeasonForm)
      setEditingSeason(false)
      await loadInitial()
      await loadDetail(selectedId)
    } catch (e) {
      setError(String(e))
    }
  }

  async function addBlackout(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) return
    try {
      await seasonBlackoutsApi.create(selectedId, newBlackout)
      setNewBlackout('')
      const b = await seasonBlackoutsApi.list(selectedId)
      setBlackouts(b)
    } catch (e) {
      setError(String(e))
    }
  }

  async function removeBlackout(bid: string) {
    if (!selectedId) return
    try {
      await seasonBlackoutsApi.delete(selectedId, bid)
      setBlackouts(prev => prev.filter(b => b.id !== bid))
    } catch (e) {
      setError(String(e))
    }
  }

  async function addPreferred(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) return
    try {
      await preferredDatesApi.create(selectedId, newPreferred.date, newPreferred.weight)
      setNewPreferred({ date: '', weight: 1.0 })
      const p = await preferredDatesApi.list(selectedId)
      setPreferred(p)
    } catch (e) {
      setError(String(e))
    }
  }

  async function removePreferred(pid: string) {
    if (!selectedId) return
    try {
      await preferredDatesApi.delete(selectedId, pid)
      setPreferred(prev => prev.filter(p => p.id !== pid))
    } catch (e) {
      setError(String(e))
    }
  }

  function getConstraintDefaultParams(type: string): Record<string, unknown> {
    return CONSTRAINT_TYPES[type]?.defaultParams ?? {}
  }

  function handleConstraintTypeChange(type: string) {
    setNewConstraintType(type)
    setNewConstraintParams(getConstraintDefaultParams(type))
  }

  async function addConstraint(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) return
    const def = CONSTRAINT_TYPES[newConstraintType]
    const params = Object.keys(newConstraintParams).length > 0
      ? newConstraintParams
      : def?.defaultParams ?? {}
    try {
      await constraintsApi.create(selectedId, {
        type: newConstraintType,
        params,
        is_hard: def?.isHard ?? true,
        weight: def?.defaultWeight ?? 1,
      })
      setNewConstraintParams(getConstraintDefaultParams(newConstraintType))
      const c = await constraintsApi.list(selectedId)
      setConstraints(c)
    } catch (e) {
      setError(String(e))
    }
  }

  async function removeConstraint(cid: string) {
    if (!selectedId) return
    try {
      await constraintsApi.delete(selectedId, cid)
      setConstraints(prev => prev.filter(c => c.id !== cid))
    } catch (e) {
      setError(String(e))
    }
  }

  async function startGeneration() {
    if (!selectedId) return
    try {
      setGenStatus(null)
      const { run_id } = await generationApi.start(selectedId)
      setGenRunId(run_id)
      // Update season status in list
      setSeasons(prev => prev.map(s => s.id === selectedId ? { ...s, status: 'generating' } : s))
      // Start polling
      if (pollRef.current) clearInterval(pollRef.current)
      pollRef.current = setInterval(async () => {
        try {
          const status = await generationApi.getStatus(selectedId, run_id)
          setGenStatus(status)
          if (status.status === 'success' || status.status === 'failed') {
            if (pollRef.current) clearInterval(pollRef.current)
            pollRef.current = null
            // Reload seasons to get updated status
            const s = await seasonsApi.list()
            setSeasons(s)
          }
        } catch (e) {
          setError(String(e))
          if (pollRef.current) clearInterval(pollRef.current)
          pollRef.current = null
        }
      }, 3000)
    } catch (e) {
      setError(String(e))
    }
  }

  const selectedSeason = seasons.find(s => s.id === selectedId)
  const divisionMap: Record<string, string> = {}
  for (const d of divisions) divisionMap[d.id] = d.name

  const inputStyle: React.CSSProperties = { padding: '0.4rem', borderRadius: 4, border: '1px solid #ccc' }
  const rowStyle: React.CSSProperties = { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0.4rem 0', borderBottom: '1px solid #f0f0f0', fontSize: '0.9rem' }
  const subheadStyle: React.CSSProperties = { marginTop: '1.25rem', marginBottom: '0.5rem', fontSize: '1rem', color: '#1a5276' }

  if (loading) return <div className="card"><p>Loading...</p></div>

  return (
    <div>
      <h1>Seasons</h1>
      {error && (
        <div className="card" style={{ background: '#fdd' }}>
          <strong>Error:</strong> {error}
          <button onClick={() => setError(null)} style={{ marginLeft: '1rem', cursor: 'pointer', background: 'none', border: 'none', fontWeight: 'bold' }}>✕</button>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '320px 1fr', gap: '1.5rem', alignItems: 'start' }}>
        {/* Left column: season list + create form */}
        <div>
          <div className="card">
            <h2 style={{ marginTop: 0 }}>Create Season</h2>
            <form onSubmit={createSeason} style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              <label>
                Name<br />
                <input
                  value={newSeason.name}
                  onChange={e => setNewSeason(p => ({ ...p, name: e.target.value }))}
                  required
                  placeholder="e.g. Spring 2026"
                  style={{ ...inputStyle, width: '100%' }}
                />
              </label>
              <fieldset style={{ border: '1px solid #ccc', borderRadius: 4, padding: '0.4rem 0.75rem' }}>
                <legend style={{ fontSize: '0.9rem' }}>Divisions (select at least one)</legend>
                {divisions.length === 0 && (
                  <p style={{ color: '#888', fontSize: '0.85rem', margin: '0.25rem 0' }}>No divisions available.</p>
                )}
                {divisions.map(d => (
                  <label key={d.id} style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.2rem', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={newSeason.division_ids.includes(d.id)}
                      onChange={e => {
                        if (e.target.checked) {
                          setNewSeason(p => ({ ...p, division_ids: [...p.division_ids, d.id] }))
                        } else {
                          setNewSeason(p => ({ ...p, division_ids: p.division_ids.filter(id => id !== d.id) }))
                        }
                      }}
                    />
                    {d.name} ({d.season_year})
                  </label>
                ))}
              </fieldset>
              <label>
                Start Date<br />
                <input
                  type="date"
                  value={newSeason.start_date}
                  onChange={e => setNewSeason(p => ({ ...p, start_date: e.target.value }))}
                  required
                  style={{ ...inputStyle, width: '100%' }}
                />
              </label>
              <label>
                End Date<br />
                <input
                  type="date"
                  value={newSeason.end_date}
                  onChange={e => setNewSeason(p => ({ ...p, end_date: e.target.value }))}
                  required
                  style={{ ...inputStyle, width: '100%' }}
                />
              </label>
              <button type="submit" className="btn btn-primary">Create Season</button>
            </form>
          </div>

          <div className="card">
            <h2 style={{ marginTop: 0 }}>All Seasons</h2>
            {seasons.length === 0 && (
              <p className="placeholder">No seasons yet.</p>
            )}
            {seasons.map(s => {
              const sc = STATUS_COLORS[s.status] || STATUS_COLORS.draft
              const isSelected = selectedId === s.id
              return (
                <div
                  key={s.id}
                  onClick={() => setSelectedId(s.id)}
                  style={{
                    padding: '0.6rem 0.75rem',
                    marginBottom: '0.4rem',
                    borderRadius: 6,
                    cursor: 'pointer',
                    border: isSelected ? '2px solid #1a5276' : '2px solid transparent',
                    background: isSelected ? '#f0f4f8' : '#fafafa',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <strong style={{ fontSize: '0.95rem' }}>{s.name}</strong>
                    <div style={{ display: 'flex', gap: '0.3rem', alignItems: 'center' }}>
                      {s.is_current && (
                        <span style={{ background: '#d4edda', color: '#155724', padding: '1px 8px', borderRadius: 10, fontSize: '0.78rem', fontWeight: 600 }}>
                          current
                        </span>
                      )}
                      <span style={{ background: sc.bg, color: sc.color, padding: '1px 8px', borderRadius: 10, fontSize: '0.78rem', fontWeight: 600 }}>
                        {s.status}
                      </span>
                    </div>
                  </div>
                  <div style={{ fontSize: '0.8rem', color: '#666', marginTop: 2 }}>
                    {(s.division_ids || []).map(id => divisionMap[id] || id).join(', ') || '—'} &middot; {s.start_date} &rarr; {s.end_date}
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Right column: season detail */}
        <div>
          {!selectedSeason ? (
            <div className="card">
              <p className="placeholder">Select a season on the left to view and configure it.</p>
            </div>
          ) : (
            <>
              <div className="card">
                {editingSeason ? (
                  <form onSubmit={saveEditSeason} style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                    <label>
                      Name<br />
                      <input
                        value={editSeasonForm.name}
                        onChange={e => setEditSeasonForm(p => ({ ...p, name: e.target.value }))}
                        required
                        style={{ ...inputStyle, width: '100%' }}
                      />
                    </label>
                    <fieldset style={{ border: '1px solid #ccc', borderRadius: 4, padding: '0.4rem 0.75rem' }}>
                      <legend style={{ fontSize: '0.9rem' }}>Divisions</legend>
                      {divisions.map(d => (
                        <label key={d.id} style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.2rem', cursor: 'pointer' }}>
                          <input
                            type="checkbox"
                            checked={editSeasonForm.division_ids.includes(d.id)}
                            onChange={e => {
                              if (e.target.checked) {
                                setEditSeasonForm(p => ({ ...p, division_ids: [...p.division_ids, d.id] }))
                              } else {
                                setEditSeasonForm(p => ({ ...p, division_ids: p.division_ids.filter(id => id !== d.id) }))
                              }
                            }}
                          />
                          {d.name} ({d.season_year})
                        </label>
                      ))}
                    </fieldset>
                    <div style={{ display: 'flex', gap: '0.5rem' }}>
                      <label style={{ flex: 1 }}>
                        Start Date<br />
                        <input
                          type="date"
                          value={editSeasonForm.start_date}
                          onChange={e => setEditSeasonForm(p => ({ ...p, start_date: e.target.value }))}
                          required
                          style={{ ...inputStyle, width: '100%' }}
                        />
                      </label>
                      <label style={{ flex: 1 }}>
                        End Date<br />
                        <input
                          type="date"
                          value={editSeasonForm.end_date}
                          onChange={e => setEditSeasonForm(p => ({ ...p, end_date: e.target.value }))}
                          required
                          style={{ ...inputStyle, width: '100%' }}
                        />
                      </label>
                    </div>
                    <div style={{ display: 'flex', gap: '0.5rem' }}>
                      <button type="submit" className="btn btn-primary" style={{ fontSize: '0.85rem' }}>Save</button>
                      <button type="button" onClick={() => setEditingSeason(false)} className="btn" style={{ fontSize: '0.85rem' }}>Cancel</button>
                    </div>
                  </form>
                ) : (
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                        <h2 style={{ margin: 0 }}>{selectedSeason.name}</h2>
                        {selectedSeason.is_current && (
                          <span style={{ background: '#d4edda', color: '#155724', padding: '1px 8px', borderRadius: 10, fontSize: '0.78rem', fontWeight: 600 }}>
                            current
                          </span>
                        )}
                      </div>
                      <div style={{ color: '#666', fontSize: '0.9rem', marginTop: 4 }}>
                        {(selectedSeason.division_ids || []).map(id => divisionMap[id] || id).join(', ') || '—'}
                        {' '}&middot;{' '}
                        {selectedSeason.start_date} &rarr; {selectedSeason.end_date}
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                      {!selectedSeason.is_current && (
                        <button onClick={() => setCurrentSeason(selectedSeason.id)} className="btn" style={{ fontSize: '0.85rem' }}>Set as Current</button>
                      )}
                      <button onClick={startEditSeason} className="btn" style={{ fontSize: '0.85rem' }}>Edit</button>
                      <Link
                        to={`/schedule?season=${selectedSeason.id}`}
                        className="btn btn-primary"
                        style={{ fontSize: '0.85rem', textDecoration: 'none' }}
                      >
                        View Schedule
                      </Link>
                      <button onClick={() => deleteSeason(selectedSeason.id)} className="btn btn-danger" style={{ fontSize: '0.85rem' }}>
                        Delete Season
                      </button>
                    </div>
                  </div>
                )}

                {/* Generation */}
                <div style={{ marginTop: '1rem', paddingTop: '1rem', borderTop: '1px solid #eee' }}>
                  <h3 style={{ ...subheadStyle, marginTop: 0 }}>Schedule Generation</h3>
                  <div style={{ display: 'flex', gap: '1rem', alignItems: 'center', flexWrap: 'wrap' }}>
                    <button
                      onClick={startGeneration}
                      className="btn btn-primary"
                      disabled={selectedSeason.status === 'generating'}
                    >
                      {selectedSeason.status === 'generating' ? 'Generating…' : 'Generate Schedule'}
                    </button>
                    {genStatus && (
                      <div style={{ fontSize: '0.9rem' }}>
                        Run status:{' '}
                        <strong style={{
                          color: genStatus.status === 'success' ? '#155724'
                               : genStatus.status === 'failed' ? '#721c24'
                               : '#856404'
                        }}>
                          {genStatus.status}
                        </strong>
                        {genStatus.status === 'failed' && genStatus.error_message && (
                          <span style={{ color: '#721c24', marginLeft: '0.5rem' }}>{genStatus.error_message}</span>
                        )}
                        {genStatus.status === 'success' && (
                          <>
                            <span style={{ marginLeft: '0.5rem', color: '#155724' }}>
                              Schedule generated successfully!
                              {genStatus.solver_stats?.num_games_scheduled !== undefined && (
                                <> {genStatus.solver_stats.num_games_scheduled as number} games scheduled.</>
                              )}
                            </span>
                            {genStatus.error_message && (
                              <div style={{ color: '#856404', background: '#fff3cd', padding: '0.4rem 0.6rem', borderRadius: 4, marginTop: '0.4rem', border: '1px solid #ffc107' }}>
                                ⚠ {genStatus.error_message}
                              </div>
                            )}
                          </>
                        )}
                      </div>
                    )}
                    {genRunId && !genStatus && (
                      <span style={{ color: '#856404', fontSize: '0.9rem' }}>Starting generation…</span>
                    )}
                  </div>
                </div>
              </div>

              {detailLoading ? (
                <div className="card"><p>Loading details…</p></div>
              ) : (
                <>
                  {/* Blackout Dates */}
                  <div className="card">
                    <h3 style={{ ...subheadStyle, marginTop: 0 }}>Season Blackout Dates</h3>
                    {blackouts.length === 0 && (
                      <p style={{ color: '#888', fontStyle: 'italic', fontSize: '0.9rem' }}>No blackout dates.</p>
                    )}
                    {blackouts.map(b => (
                      <div key={b.id} style={rowStyle}>
                        <span>{b.blackout_date}</span>
                        <button onClick={() => removeBlackout(b.id)} className="btn btn-danger" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Delete</button>
                      </div>
                    ))}
                    <form onSubmit={addBlackout} style={{ display: 'flex', gap: '0.5rem', marginTop: '0.75rem', alignItems: 'flex-end' }}>
                      <label>
                        Date<br />
                        <input
                          type="date"
                          value={newBlackout}
                          onChange={e => setNewBlackout(e.target.value)}
                          required
                          style={inputStyle}
                        />
                      </label>
                      <button type="submit" className="btn btn-primary">Add Blackout</button>
                    </form>
                  </div>

                  {/* Preferred Interleague Dates */}
                  <div className="card">
                    <h3 style={{ ...subheadStyle, marginTop: 0 }}>Preferred Interleague Dates</h3>
                    {preferred.length === 0 && (
                      <p style={{ color: '#888', fontStyle: 'italic', fontSize: '0.9rem' }}>No preferred dates.</p>
                    )}
                    {preferred.map(p => (
                      <div key={p.id} style={rowStyle}>
                        <span>
                          {p.preferred_date}
                          <span style={{ color: '#666', marginLeft: '0.5rem', fontSize: '0.85rem' }}>weight: {p.weight}</span>
                        </span>
                        <button onClick={() => removePreferred(p.id)} className="btn btn-danger" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Delete</button>
                      </div>
                    ))}
                    <form onSubmit={addPreferred} style={{ display: 'flex', gap: '0.5rem', marginTop: '0.75rem', alignItems: 'flex-end' }}>
                      <label>
                        Date<br />
                        <input
                          type="date"
                          value={newPreferred.date}
                          onChange={e => setNewPreferred(p => ({ ...p, date: e.target.value }))}
                          required
                          style={inputStyle}
                        />
                      </label>
                      <label>
                        Weight<br />
                        <input
                          type="number"
                          step="0.1"
                          min="0"
                          value={newPreferred.weight}
                          onChange={e => setNewPreferred(p => ({ ...p, weight: Number(e.target.value) }))}
                          style={{ ...inputStyle, width: 80 }}
                        />
                      </label>
                      <button type="submit" className="btn btn-primary">Add Date</button>
                    </form>
                  </div>

                  {/* Division Games Required */}
                  {(selectedSeason.division_ids || []).length > 0 && (
                    <div className="card">
                      <h3 style={{ ...subheadStyle, marginTop: 0 }}>Games Required per Division</h3>
                      <p style={{ color: '#666', fontSize: '0.85rem', marginTop: 0 }}>
                        Target number of games each team in a division should receive. Defaults to 20.
                      </p>
                      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                        <thead>
                          <tr style={{ borderBottom: '2px solid #eee', textAlign: 'left' }}>
                            <th style={{ padding: '0.3rem 0.5rem', fontSize: '0.9rem' }}>Division</th>
                            <th style={{ padding: '0.3rem 0.5rem', fontSize: '0.9rem' }}>Games Required</th>
                            <th style={{ padding: '0.3rem 0.5rem' }}></th>
                          </tr>
                        </thead>
                        <tbody>
                          {(selectedSeason.division_ids || []).map(divId => {
                            const existing = divGamesReq.find(d => d.division_id === divId)
                            const current = existing?.games_required ?? 20
                            const editing = editingDivGamesReq[divId]
                            const isEditing = editing !== undefined
                            return (
                              <tr key={divId} style={{ borderBottom: '1px solid #f0f0f0' }}>
                                <td style={{ padding: '0.4rem 0.5rem', fontSize: '0.9rem' }}>
                                  {divisionMap[divId] || divId}
                                </td>
                                <td style={{ padding: '0.4rem 0.5rem' }}>
                                  {isEditing ? (
                                    <input
                                      type="number"
                                      min="1"
                                      value={editing}
                                      onChange={e => setEditingDivGamesReq(prev => ({ ...prev, [divId]: Number(e.target.value) }))}
                                      style={{ ...inputStyle, width: 80 }}
                                      autoFocus
                                    />
                                  ) : (
                                    <span>{current}</span>
                                  )}
                                </td>
                                <td style={{ padding: '0.4rem 0.5rem', display: 'flex', gap: '0.25rem' }}>
                                  {isEditing ? (
                                    <>
                                      <button onClick={() => saveDivGamesReq(divId)} className="btn btn-primary" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Save</button>
                                      <button onClick={() => setEditingDivGamesReq(prev => { const n = { ...prev }; delete n[divId]; return n })} className="btn" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Cancel</button>
                                    </>
                                  ) : (
                                    <button onClick={() => setEditingDivGamesReq(prev => ({ ...prev, [divId]: current }))} className="btn btn-primary" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Edit</button>
                                  )}
                                </td>
                              </tr>
                            )
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {/* Constraints */}
                  <div className="card">
                    <h3 style={{ ...subheadStyle, marginTop: 0 }}>Constraints</h3>
                    {constraints.length === 0 && (
                      <p style={{ color: '#888', fontStyle: 'italic', fontSize: '0.9rem' }}>No constraints configured.</p>
                    )}
                    {constraints.map(c => {
                      const def = CONSTRAINT_TYPES[c.type]
                      const isAutoInjected = c.params.auto_injected === true
                      const displayParams = Object.entries(c.params).filter(([k]) => k !== 'auto_injected')
                      return (
                        <div key={c.id} style={{ ...rowStyle, flexWrap: 'wrap', gap: '0.25rem', opacity: isAutoInjected ? 0.65 : 1 }}>
                          <div>
                            <strong style={{ fontSize: '0.9rem' }}>{def?.label || c.type}</strong>
                            {isAutoInjected && (
                              <span style={{ marginLeft: '0.4rem', background: '#e9ecef', color: '#555', padding: '1px 6px', borderRadius: 10, fontSize: '0.72rem' }} title="Auto-injected based on division/team settings">
                                auto
                              </span>
                            )}
                            <span style={{
                              marginLeft: '0.5rem',
                              background: c.is_hard ? '#fde8d8' : '#e8f4d8',
                              color: c.is_hard ? '#8b3a0a' : '#2e5e0a',
                              padding: '1px 7px',
                              borderRadius: 10,
                              fontSize: '0.75rem',
                            }}>
                              {c.is_hard ? 'hard' : `soft w=${c.weight}`}
                            </span>
                            {displayParams.length > 0 && (
                              <span style={{ marginLeft: '0.5rem', fontSize: '0.8rem', color: '#555' }}>
                                {displayParams.map(([k, v]) => `${k}: ${JSON.stringify(v)}`).join(', ')}
                              </span>
                            )}
                          </div>
                          {!isAutoInjected && (
                            <button onClick={() => removeConstraint(c.id)} className="btn btn-danger" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Delete</button>
                          )}
                        </div>
                      )
                    })}

                    <form onSubmit={addConstraint} style={{ marginTop: '1rem', background: '#f9f9f9', padding: '0.75rem', borderRadius: 6 }}>
                      <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
                        <label>
                          Constraint Type<br />
                          <select
                            value={newConstraintType}
                            onChange={e => handleConstraintTypeChange(e.target.value)}
                            style={inputStyle}
                          >
                            {Object.entries(CONSTRAINT_TYPES).map(([key, def]) => (
                              <option key={key} value={key}>{def.label}</option>
                            ))}
                          </select>
                        </label>

                        {(CONSTRAINT_TYPES[newConstraintType]?.paramFields || []).map(field => (
                          <label key={field.key}>
                            {field.label}<br />
                            <input
                              type={field.type}
                              value={String(newConstraintParams[field.key] ?? CONSTRAINT_TYPES[newConstraintType]?.defaultParams[field.key] ?? '')}
                              onChange={e => setNewConstraintParams(prev => ({
                                ...prev,
                                [field.key]: field.type === 'number' ? Number(e.target.value) : e.target.value,
                              }))}
                              style={{ ...inputStyle, width: 100 }}
                            />
                          </label>
                        ))}

                        <button type="submit" className="btn btn-primary">Add Constraint</button>
                      </div>
                      <div style={{ marginTop: '0.4rem', fontSize: '0.8rem', color: '#666' }}>
                        {CONSTRAINT_TYPES[newConstraintType]?.isHard
                          ? 'Hard constraint (must be satisfied)'
                          : `Soft constraint (weight: ${CONSTRAINT_TYPES[newConstraintType]?.defaultWeight ?? 1})`
                        }
                      </div>
                    </form>
                  </div>
                </>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
