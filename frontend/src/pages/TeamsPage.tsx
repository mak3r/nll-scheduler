import { useState, useEffect } from 'react'
import { divisionsApi, teamsApiClient, type Division, type Team, type DivisionFieldRule, type MatchupRule } from '../api/teams'
import { fieldsApiClient, type Field } from '../api/fields'
import { seasonsApi, type Season } from '../api/schedule'

export default function TeamsPage() {
  const [divisions, setDivisions] = useState<Division[]>([])
  const [teams, setTeams] = useState<Record<string, Team[]>>({})
  const [allFields, setAllFields] = useState<Field[]>([])
  const [fieldRules, setFieldRules] = useState<Record<string, DivisionFieldRule[]>>({})
  const [newRuleForms, setNewRuleForms] = useState<Record<string, { fieldId: string; ruleType: 'allowed' | 'preferred' }>>({})
  const [matchupRules, setMatchupRules] = useState<Record<string, MatchupRule[]>>({})
  const [newMatchupForms, setNewMatchupForms] = useState<Record<string, { teamAId: string; teamBId: string; minGames: number; maxGames: number }>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [seasons, setSeasons] = useState<Season[]>([])

  const [newDivName, setNewDivName] = useState('')
  const [newDivSeasonId, setNewDivSeasonId] = useState('')

  const [newTeamForms, setNewTeamForms] = useState<Record<string, {
    name: string; shortCode: string; teamType: string
  }>>({})

  const [bulkForms, setBulkForms] = useState<Record<string, { text: string; teamType: string; submitting: boolean }>>({})
  const [editingTeamId, setEditingTeamId] = useState<string | null>(null)
  const [editTeamForm, setEditTeamForm] = useState({ name: '', shortCode: '', teamType: 'local', divisionId: '', homeFieldId: undefined as string | undefined })

  useEffect(() => { loadData() }, [])

  async function loadData() {
    setLoading(true)
    setError(null)
    try {
      const [divs, fields, seasonList] = await Promise.all([
        divisionsApi.list(),
        fieldsApiClient.list(),
        seasonsApi.list(),
      ])
      setSeasons(seasonList)
      // Pre-select the current season if not already set
      setNewDivSeasonId(prev => {
        if (prev) return prev
        const current = seasonList.find(s => s.is_current)
        return current?.id ?? (seasonList[0]?.id ?? '')
      })
      setDivisions(divs)
      setAllFields(fields.filter(f => f.is_active))
      const teamsMap: Record<string, Team[]> = {}
      const rulesMap: Record<string, DivisionFieldRule[]> = {}
      const matchupMap: Record<string, MatchupRule[]> = {}
      await Promise.all(divs.map(async (d) => {
        try {
          const result = await divisionsApi.getTeamsWithRules(d.id)
          teamsMap[d.id] = result.teams
          matchupMap[d.id] = result.matchup_rules
        } catch {
          teamsMap[d.id] = []
          matchupMap[d.id] = []
        }
        try {
          rulesMap[d.id] = await divisionsApi.listFieldRules(d.id)
        } catch {
          rulesMap[d.id] = []
        }
      }))
      setTeams(teamsMap)
      setFieldRules(rulesMap)
      setMatchupRules(matchupMap)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function createDivision(e: React.FormEvent) {
    e.preventDefault()
    try {
      const selectedSeason = seasons.find(s => s.id === newDivSeasonId)
      const seasonYear = selectedSeason
        ? new Date(selectedSeason.start_date).getFullYear()
        : new Date().getFullYear()
      const d = await divisionsApi.create({ name: newDivName, season_year: seasonYear, season_id: newDivSeasonId })
      // Also link this division to the season in schedule-service
      if (newDivSeasonId && selectedSeason) {
        const updatedIds = [...(selectedSeason.division_ids || []), d.id]
        await seasonsApi.update(newDivSeasonId, { division_ids: updatedIds })
      }
      setNewDivName('')
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  async function deleteDivision(id: string) {
    if (!confirm('Delete this division and all its teams?')) return
    try {
      await divisionsApi.delete(id)
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  function getTeamForm(divId: string) {
    return newTeamForms[divId] || { name: '', shortCode: '', teamType: 'local' }
  }

  function setTeamForm(divId: string, updates: Partial<ReturnType<typeof getTeamForm>>) {
    setNewTeamForms(prev => ({ ...prev, [divId]: { ...getTeamForm(divId), ...updates } }))
  }

  async function createTeam(e: React.FormEvent, divId: string) {
    e.preventDefault()
    const form = getTeamForm(divId)
    try {
      await teamsApiClient.create({
        division_id: divId,
        name: form.name,
        short_code: form.shortCode,
        team_type: form.teamType as 'local' | 'interleague',
      })
      setNewTeamForms(prev => { const n = { ...prev }; delete n[divId]; return n })
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  async function deleteTeam(id: string) {
    if (!confirm('Delete this team?')) return
    try {
      await teamsApiClient.delete(id)
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  function getBulkForm(divId: string) {
    return bulkForms[divId] || { text: '', teamType: 'local', submitting: false }
  }

  async function submitBulkAdd(e: React.FormEvent, divId: string) {
    e.preventDefault()
    const form = getBulkForm(divId)
    const names = form.text.split(/[\n,]+/).map(s => s.trim()).filter(Boolean)
    if (names.length === 0) return
    setBulkForms(prev => ({ ...prev, [divId]: { ...getBulkForm(divId), submitting: true } }))
    try {
      for (const name of names) {
        const initials = name.split(/\s+/).map(w => w[0]?.toUpperCase() ?? '').join('').slice(0, 4) || name.slice(0, 3).toUpperCase()
        await teamsApiClient.create({
          division_id: divId,
          name,
          short_code: initials,
          team_type: form.teamType as 'local' | 'interleague',
        })
      }
      setBulkForms(prev => { const n = { ...prev }; delete n[divId]; return n })
      await loadData()
    } catch (e) {
      setError(String(e))
      setBulkForms(prev => ({ ...prev, [divId]: { ...getBulkForm(divId), submitting: false } }))
    }
  }

  function startEditTeam(team: Team) {
    setEditingTeamId(team.id)
    setEditTeamForm({
      name: team.name,
      shortCode: team.short_code,
      teamType: team.team_type,
      divisionId: team.division_id,
      homeFieldId: team.home_field_id,
    })
  }

  async function saveTeam(id: string) {
    try {
      await teamsApiClient.update(id, {
        division_id: editTeamForm.divisionId,
        name: editTeamForm.name,
        short_code: editTeamForm.shortCode,
        team_type: editTeamForm.teamType as 'local' | 'interleague',
        home_field_id: editTeamForm.homeFieldId,
      })
      setEditingTeamId(null)
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  function getNewRuleForm(divId: string) {
    return newRuleForms[divId] || { fieldId: '', ruleType: 'allowed' as const }
  }

  async function addFieldRule(e: React.FormEvent, divId: string) {
    e.preventDefault()
    const form = getNewRuleForm(divId)
    if (!form.fieldId) return
    try {
      await divisionsApi.createFieldRule(divId, { field_id: form.fieldId, rule_type: form.ruleType })
      setNewRuleForms(prev => { const n = { ...prev }; delete n[divId]; return n })
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  async function removeFieldRule(divId: string, ruleId: string) {
    try {
      await divisionsApi.deleteFieldRule(divId, ruleId)
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  function getMatchupForm(divId: string) {
    return newMatchupForms[divId] || { teamAId: '', teamBId: '', minGames: 2, maxGames: 2 }
  }

  async function addMatchupRule(e: React.FormEvent, divId: string) {
    e.preventDefault()
    const form = getMatchupForm(divId)
    if (!form.teamAId || !form.teamBId || form.teamAId === form.teamBId) return
    try {
      await teamsApiClient.createMatchupRule(form.teamAId, {
        team_b_id: form.teamBId,
        min_games: form.minGames,
        max_games: form.maxGames,
      })
      setNewMatchupForms(prev => { const n = { ...prev }; delete n[divId]; return n })
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  async function removeMatchupRule(teamAId: string, ruleId: string) {
    try {
      await teamsApiClient.deleteMatchupRule(teamAId, ruleId)
      await loadData()
    } catch (e) {
      setError(String(e))
    }
  }

  const inputStyle: React.CSSProperties = {
    padding: '0.4rem',
    borderRadius: 4,
    border: '1px solid #ccc',
  }

  if (loading) return <div className="card"><p>Loading...</p></div>

  return (
    <div>
      <h1>Teams</h1>
      {error && (
        <div className="card" style={{ background: '#fdd' }}>
          <strong>Error:</strong> {error}
          <button onClick={() => setError(null)} style={{ marginLeft: '1rem', cursor: 'pointer', background: 'none', border: 'none', fontWeight: 'bold' }}>✕</button>
        </div>
      )}

      <div className="card">
        <h2>Add Division</h2>
        <form onSubmit={createDivision} style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <div>
            <label>
              Name<br />
              <input
                value={newDivName}
                onChange={e => setNewDivName(e.target.value)}
                required
                placeholder="e.g. Majors"
                style={inputStyle}
              />
            </label>
          </div>
          <div>
            <label>
              Season<br />
              <select
                value={newDivSeasonId}
                onChange={e => setNewDivSeasonId(e.target.value)}
                required
                style={inputStyle}
              >
                <option value="">— Select a season —</option>
                {seasons.map(s => (
                  <option key={s.id} value={s.id}>
                    {s.name}{s.is_current ? ' (current)' : ''}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <button type="submit" className="btn btn-primary">Add Division</button>
        </form>
      </div>

      {divisions.map(div => (
        <div key={div.id} className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h2 style={{ margin: 0 }}>{div.name} <span style={{ color: '#888', fontWeight: 400, fontSize: '0.9rem' }}>({div.season_year})</span></h2>
            <button onClick={() => deleteDivision(div.id)} className="btn btn-danger">Delete Division</button>
          </div>

          <table style={{ width: '100%', borderCollapse: 'collapse', margin: '1rem 0' }}>
            <thead>
              <tr style={{ borderBottom: '2px solid #eee', textAlign: 'left' }}>
                <th style={{ padding: '0.4rem' }}>Name</th>
                <th style={{ padding: '0.4rem' }}>Code</th>
                <th style={{ padding: '0.4rem' }}>Type</th>
                <th style={{ padding: '0.4rem' }}></th>
              </tr>
            </thead>
            <tbody>
              {(teams[div.id] || []).length === 0 ? (
                <tr>
                  <td colSpan={4} style={{ padding: '0.75rem 0.4rem', color: '#888', fontStyle: 'italic' }}>
                    No teams yet. Add one below.
                  </td>
                </tr>
              ) : (
                (teams[div.id] || []).map(team => (
                  editingTeamId === team.id ? (
                    <tr key={team.id} style={{ borderBottom: '1px solid #eee', background: '#f9f9f9' }}>
                      <td style={{ padding: '0.4rem' }}>
                        <input value={editTeamForm.name} onChange={e => setEditTeamForm(p => ({ ...p, name: e.target.value }))} style={inputStyle} />
                      </td>
                      <td style={{ padding: '0.4rem' }}>
                        <input value={editTeamForm.shortCode} onChange={e => setEditTeamForm(p => ({ ...p, shortCode: e.target.value }))} style={{ ...inputStyle, width: 60 }} />
                      </td>
                      <td style={{ padding: '0.4rem' }}>
                        <select value={editTeamForm.teamType} onChange={e => setEditTeamForm(p => ({ ...p, teamType: e.target.value }))} style={inputStyle}>
                          <option value="local">Local</option>
                          <option value="interleague">Interleague</option>
                        </select>
                      </td>
                      <td style={{ padding: '0.4rem', display: 'flex', gap: '0.25rem' }}>
                        <button onClick={() => saveTeam(team.id)} className="btn btn-primary" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Save</button>
                        <button onClick={() => setEditingTeamId(null)} className="btn" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Cancel</button>
                      </td>
                    </tr>
                  ) : (
                    <tr key={team.id} style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '0.4rem' }}>{team.name}</td>
                      <td style={{ padding: '0.4rem' }}><code>{team.short_code}</code></td>
                      <td style={{ padding: '0.4rem' }}>
                        <span style={{
                          background: team.team_type === 'interleague' ? '#e8f4f8' : '#f0f8e8',
                          color: team.team_type === 'interleague' ? '#1a5276' : '#1a6b1a',
                          padding: '2px 8px',
                          borderRadius: 12,
                          fontSize: '0.85rem',
                        }}>
                          {team.team_type}
                        </span>
                      </td>
                      <td style={{ padding: '0.4rem', display: 'flex', gap: '0.25rem' }}>
                        <button onClick={() => startEditTeam(team)} className="btn btn-primary" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Edit</button>
                        <button onClick={() => deleteTeam(team.id)} className="btn btn-danger" style={{ fontSize: '0.8rem', padding: '2px 10px' }}>Delete</button>
                      </td>
                    </tr>
                  )
                ))
              )}
            </tbody>
          </table>

          <form
            onSubmit={e => createTeam(e, div.id)}
            style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end', borderTop: '1px solid #eee', paddingTop: '1rem' }}
          >
            <label>
              Team Name<br />
              <input
                value={getTeamForm(div.id).name}
                onChange={e => setTeamForm(div.id, { name: e.target.value })}
                required
                placeholder="e.g. Red Sox"
                style={inputStyle}
              />
            </label>
            <label>
              Short Code<br />
              <input
                value={getTeamForm(div.id).shortCode}
                onChange={e => setTeamForm(div.id, { shortCode: e.target.value })}
                required
                placeholder="RSX"
                style={{ ...inputStyle, width: 70 }}
              />
            </label>
            <label>
              Type<br />
              <select
                value={getTeamForm(div.id).teamType}
                onChange={e => setTeamForm(div.id, { teamType: e.target.value })}
                style={inputStyle}
              >
                <option value="local">Local</option>
                <option value="interleague">Interleague</option>
              </select>
            </label>
            <button type="submit" className="btn btn-primary">Add Team</button>
          </form>

          {/* Bulk Add Teams */}
          <div style={{ borderTop: '1px solid #eee', paddingTop: '1rem', marginTop: '1rem' }}>
            <h3 style={{ margin: '0 0 0.5rem', fontSize: '1rem' }}>Bulk Add Teams</h3>
            <form onSubmit={e => submitBulkAdd(e, div.id)} style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
              <label>
                Team Names (one per line or comma-separated)<br />
                <textarea
                  value={getBulkForm(div.id).text}
                  onChange={e => setBulkForms(prev => ({ ...prev, [div.id]: { ...getBulkForm(div.id), text: e.target.value } }))}
                  placeholder={'Red Sox\nBlue Jays\nYankees'}
                  rows={3}
                  style={{ ...inputStyle, width: 220, fontFamily: 'inherit', resize: 'vertical' }}
                />
              </label>
              <label>
                Type<br />
                <select
                  value={getBulkForm(div.id).teamType}
                  onChange={e => setBulkForms(prev => ({ ...prev, [div.id]: { ...getBulkForm(div.id), teamType: e.target.value } }))}
                  style={inputStyle}
                >
                  <option value="local">Local</option>
                  <option value="interleague">Interleague</option>
                </select>
              </label>
              <div>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={getBulkForm(div.id).submitting || !getBulkForm(div.id).text.trim()}
                >
                  {getBulkForm(div.id).submitting ? 'Adding…' : 'Bulk Add'}
                </button>
                <div style={{ fontSize: '0.78rem', color: '#666', marginTop: '0.25rem' }}>
                  Short codes auto-generated from initials
                </div>
              </div>
            </form>
          </div>

          {/* Field Access Rules */}
          <div style={{ borderTop: '1px solid #eee', paddingTop: '1rem', marginTop: '1rem' }}>
            <h3 style={{ margin: '0 0 0.5rem', fontSize: '1rem' }}>Field Access Rules</h3>
            {(fieldRules[div.id] || []).length === 0 ? (
              <p style={{ color: '#888', fontStyle: 'italic', margin: '0 0 0.75rem' }}>
                No restrictions — all active fields are available.
              </p>
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '0.75rem' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid #eee', textAlign: 'left' }}>
                    <th style={{ padding: '0.3rem 0.4rem' }}>Field</th>
                    <th style={{ padding: '0.3rem 0.4rem' }}>Rule</th>
                    <th style={{ padding: '0.3rem 0.4rem' }}></th>
                  </tr>
                </thead>
                <tbody>
                  {(fieldRules[div.id] || []).map(rule => {
                    const field = allFields.find(f => f.id === rule.field_id)
                    return (
                      <tr key={rule.id} style={{ borderBottom: '1px solid #f0f0f0' }}>
                        <td style={{ padding: '0.3rem 0.4rem' }}>{field ? field.name : rule.field_id}</td>
                        <td style={{ padding: '0.3rem 0.4rem' }}>
                          <span style={{
                            background: rule.rule_type === 'preferred' ? '#fef3cd' : '#dbeafe',
                            color: rule.rule_type === 'preferred' ? '#92660a' : '#1e40af',
                            padding: '2px 8px',
                            borderRadius: 12,
                            fontSize: '0.8rem',
                          }}>
                            {rule.rule_type}
                          </span>
                        </td>
                        <td style={{ padding: '0.3rem 0.4rem' }}>
                          <button
                            onClick={() => removeFieldRule(div.id, rule.id)}
                            className="btn btn-danger"
                            style={{ fontSize: '0.75rem', padding: '2px 8px' }}
                          >
                            Remove
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
            <form onSubmit={e => addFieldRule(e, div.id)} style={{ display: 'flex', gap: '0.5rem', alignItems: 'flex-end', flexWrap: 'wrap' }}>
              <label>
                Field<br />
                <select
                  value={getNewRuleForm(div.id).fieldId}
                  onChange={e => setNewRuleForms(prev => ({ ...prev, [div.id]: { ...getNewRuleForm(div.id), fieldId: e.target.value } }))}
                  required
                  style={inputStyle}
                >
                  <option value="">Select field...</option>
                  {allFields.map(f => (
                    <option key={f.id} value={f.id}>{f.name}</option>
                  ))}
                </select>
              </label>
              <label>
                Rule Type<br />
                <select
                  value={getNewRuleForm(div.id).ruleType}
                  onChange={e => setNewRuleForms(prev => ({ ...prev, [div.id]: { ...getNewRuleForm(div.id), ruleType: e.target.value as 'allowed' | 'preferred' } }))}
                  style={inputStyle}
                >
                  <option value="allowed">allowed</option>
                  <option value="preferred">preferred</option>
                </select>
              </label>
              <button type="submit" className="btn btn-primary" style={{ fontSize: '0.85rem' }}>Add Rule</button>
            </form>
          </div>

          {/* Matchup Rules */}
          <div style={{ borderTop: '1px solid #eee', paddingTop: '1rem', marginTop: '1rem' }}>
            <h3 style={{ margin: '0 0 0.5rem', fontSize: '1rem' }}>Matchup Rules</h3>
            <p style={{ color: '#666', fontSize: '0.83rem', margin: '0 0 0.5rem' }}>
              Override how many times specific team pairs play each other. Leave empty to use the division default.
            </p>
            {(matchupRules[div.id] || []).length > 0 && (
              <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '0.75rem' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid #eee', textAlign: 'left' }}>
                    <th style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>Team A</th>
                    <th style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>Team B</th>
                    <th style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>Min</th>
                    <th style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>Max</th>
                    <th style={{ padding: '0.3rem 0.4rem' }}></th>
                  </tr>
                </thead>
                <tbody>
                  {(matchupRules[div.id] || []).map(rule => {
                    const teamA = (teams[div.id] || []).find(t => t.id === rule.team_a_id)
                    const teamB = (teams[div.id] || []).find(t => t.id === rule.team_b_id)
                    return (
                      <tr key={rule.id} style={{ borderBottom: '1px solid #f0f0f0' }}>
                        <td style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>{teamA?.name || rule.team_a_id}</td>
                        <td style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>{teamB?.name || rule.team_b_id}</td>
                        <td style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>{rule.min_games}</td>
                        <td style={{ padding: '0.3rem 0.4rem', fontSize: '0.85rem' }}>{rule.max_games}</td>
                        <td style={{ padding: '0.3rem 0.4rem' }}>
                          <button
                            onClick={() => removeMatchupRule(rule.team_a_id, rule.id)}
                            className="btn btn-danger"
                            style={{ fontSize: '0.75rem', padding: '2px 8px' }}
                          >
                            Remove
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
            <form onSubmit={e => addMatchupRule(e, div.id)} style={{ display: 'flex', gap: '0.5rem', alignItems: 'flex-end', flexWrap: 'wrap' }}>
              <label>
                Team A<br />
                <select
                  value={getMatchupForm(div.id).teamAId}
                  onChange={e => setNewMatchupForms(prev => ({ ...prev, [div.id]: { ...getMatchupForm(div.id), teamAId: e.target.value } }))}
                  required
                  style={inputStyle}
                >
                  <option value="">Select team...</option>
                  {(teams[div.id] || []).map(t => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
              </label>
              <label>
                Team B<br />
                <select
                  value={getMatchupForm(div.id).teamBId}
                  onChange={e => setNewMatchupForms(prev => ({ ...prev, [div.id]: { ...getMatchupForm(div.id), teamBId: e.target.value } }))}
                  required
                  style={inputStyle}
                >
                  <option value="">Select team...</option>
                  {(teams[div.id] || []).filter(t => t.id !== getMatchupForm(div.id).teamAId).map(t => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
              </label>
              <label>
                Min<br />
                <input
                  type="number"
                  min="1"
                  value={getMatchupForm(div.id).minGames}
                  onChange={e => setNewMatchupForms(prev => ({ ...prev, [div.id]: { ...getMatchupForm(div.id), minGames: Number(e.target.value) } }))}
                  style={{ ...inputStyle, width: 60 }}
                />
              </label>
              <label>
                Max<br />
                <input
                  type="number"
                  min="1"
                  value={getMatchupForm(div.id).maxGames}
                  onChange={e => setNewMatchupForms(prev => ({ ...prev, [div.id]: { ...getMatchupForm(div.id), maxGames: Number(e.target.value) } }))}
                  style={{ ...inputStyle, width: 60 }}
                />
              </label>
              <button type="submit" className="btn btn-primary" style={{ fontSize: '0.85rem' }}>Add Rule</button>
            </form>
          </div>
        </div>
      ))}

      {divisions.length === 0 && (
        <div className="card">
          <p className="placeholder">No divisions yet. Create one above.</p>
        </div>
      )}
    </div>
  )
}
