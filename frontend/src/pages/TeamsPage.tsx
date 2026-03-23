import { useState, useEffect } from 'react'
import { divisionsApi, teamsApiClient, type Division, type Team } from '../api/teams'

export default function TeamsPage() {
  const [divisions, setDivisions] = useState<Division[]>([])
  const [teams, setTeams] = useState<Record<string, Team[]>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [newDivName, setNewDivName] = useState('')
  const [newDivYear, setNewDivYear] = useState(new Date().getFullYear())

  const [newTeamForms, setNewTeamForms] = useState<Record<string, {
    name: string; shortCode: string; teamType: string; gamesRequired: number
  }>>({})

  useEffect(() => { loadData() }, [])

  async function loadData() {
    setLoading(true)
    setError(null)
    try {
      const divs = await divisionsApi.list()
      setDivisions(divs)
      const teamsMap: Record<string, Team[]> = {}
      await Promise.all(divs.map(async (d) => {
        try {
          const result = await divisionsApi.getTeamsWithRules(d.id)
          teamsMap[d.id] = result.teams
        } catch {
          teamsMap[d.id] = []
        }
      }))
      setTeams(teamsMap)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function createDivision(e: React.FormEvent) {
    e.preventDefault()
    try {
      await divisionsApi.create({ name: newDivName, season_year: newDivYear })
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
    return newTeamForms[divId] || { name: '', shortCode: '', teamType: 'local', gamesRequired: 20 }
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
        games_required: form.gamesRequired,
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
              Year<br />
              <input
                type="number"
                value={newDivYear}
                onChange={e => setNewDivYear(Number(e.target.value))}
                required
                style={{ ...inputStyle, width: 90 }}
              />
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
                <th style={{ padding: '0.4rem' }}>Games Required</th>
                <th style={{ padding: '0.4rem' }}></th>
              </tr>
            </thead>
            <tbody>
              {(teams[div.id] || []).length === 0 ? (
                <tr>
                  <td colSpan={5} style={{ padding: '0.75rem 0.4rem', color: '#888', fontStyle: 'italic' }}>
                    No teams yet. Add one below.
                  </td>
                </tr>
              ) : (
                (teams[div.id] || []).map(team => (
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
                    <td style={{ padding: '0.4rem' }}>{team.games_required}</td>
                    <td style={{ padding: '0.4rem' }}>
                      <button
                        onClick={() => deleteTeam(team.id)}
                        className="btn btn-danger"
                        style={{ fontSize: '0.8rem', padding: '2px 10px' }}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
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
            <label>
              Games Required<br />
              <input
                type="number"
                value={getTeamForm(div.id).gamesRequired}
                onChange={e => setTeamForm(div.id, { gamesRequired: Number(e.target.value) })}
                min={1}
                style={{ ...inputStyle, width: 70 }}
              />
            </label>
            <button type="submit" className="btn btn-primary">Add Team</button>
          </form>
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
