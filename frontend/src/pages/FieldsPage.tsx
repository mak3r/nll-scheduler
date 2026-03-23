import { useState, useEffect } from 'react'
import {
  fieldsApiClient,
  availabilityApi,
  blackoutApi,
  type Field,
  type AvailabilityWindow,
  type BlackoutDate,
} from '../api/fields'

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

interface WindowForm {
  windowType: string
  daysOfWeek: number[]
  startDate: string
  endDate: string
  startTime: string
  endTime: string
}

interface BlackoutForm {
  date: string
  reason: string
}

export default function FieldsPage() {
  const [fields, setFields] = useState<Field[]>([])
  const [windows, setWindows] = useState<Record<string, AvailabilityWindow[]>>({})
  const [blackouts, setBlackouts] = useState<Record<string, BlackoutDate[]>>({})
  const [expanded, setExpanded] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [newField, setNewField] = useState({ name: '', address: '', maxGamesPerDay: 4 })
  const [newWindow, setNewWindow] = useState<Record<string, WindowForm>>({})
  const [newBlackout, setNewBlackout] = useState<Record<string, BlackoutForm>>({})

  const [editingFieldId, setEditingFieldId] = useState<string | null>(null)
  const [editFieldForm, setEditFieldForm] = useState({ name: '', address: '', maxGamesPerDay: 4 })

  useEffect(() => { loadFields() }, [])

  async function loadFields() {
    setLoading(true)
    try {
      const f = await fieldsApiClient.list()
      setFields(f)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function loadFieldDetail(fieldId: string) {
    try {
      const [w, b] = await Promise.all([
        availabilityApi.list(fieldId),
        blackoutApi.list(fieldId),
      ])
      setWindows(prev => ({ ...prev, [fieldId]: w }))
      setBlackouts(prev => ({ ...prev, [fieldId]: b }))
    } catch (e) {
      setError(String(e))
    }
  }

  async function toggleExpand(fieldId: string) {
    if (expanded === fieldId) {
      setExpanded(null)
      return
    }
    setExpanded(fieldId)
    await loadFieldDetail(fieldId)
  }

  async function createField(e: React.FormEvent) {
    e.preventDefault()
    try {
      await fieldsApiClient.create({
        name: newField.name,
        address: newField.address || undefined,
        max_games_per_day: newField.maxGamesPerDay,
        is_active: true,
      })
      setNewField({ name: '', address: '', maxGamesPerDay: 4 })
      await loadFields()
    } catch (e) {
      setError(String(e))
    }
  }

  function startEditField(field: Field) {
    setEditingFieldId(field.id)
    setEditFieldForm({
      name: field.name,
      address: field.address ?? '',
      maxGamesPerDay: field.max_games_per_day,
    })
  }

  async function saveField(id: string) {
    try {
      await fieldsApiClient.update(id, {
        name: editFieldForm.name,
        address: editFieldForm.address || undefined,
        max_games_per_day: editFieldForm.maxGamesPerDay,
      })
      setEditingFieldId(null)
      await loadFields()
    } catch (e) {
      setError(String(e))
    }
  }

  async function deleteField(id: string) {
    if (!confirm('Delete this field and all its availability/blackout data?')) return
    try {
      await fieldsApiClient.delete(id)
      if (expanded === id) setExpanded(null)
      await loadFields()
    } catch (e) {
      setError(String(e))
    }
  }

  function getWindowForm(fid: string): WindowForm {
    return newWindow[fid] || {
      windowType: 'recurring',
      daysOfWeek: [],
      startDate: '',
      endDate: '',
      startTime: '09:00',
      endTime: '17:00',
    }
  }

  function updateWindowForm(fid: string, updates: Partial<WindowForm>) {
    setNewWindow(prev => ({ ...prev, [fid]: { ...getWindowForm(fid), ...updates } }))
  }

  function toggleDay(fid: string, day: number) {
    const form = getWindowForm(fid)
    const days = form.daysOfWeek.includes(day)
      ? form.daysOfWeek.filter(d => d !== day)
      : [...form.daysOfWeek, day]
    updateWindowForm(fid, { daysOfWeek: days })
  }

  async function createWindow(e: React.FormEvent, fid: string) {
    e.preventDefault()
    const form = getWindowForm(fid)
    try {
      await availabilityApi.create(fid, {
        window_type: form.windowType as 'recurring' | 'oneoff',
        days_of_week: form.daysOfWeek,
        start_date: form.startDate,
        end_date: form.endDate,
        start_time: form.startTime,
        end_time: form.endTime,
      })
      setNewWindow(prev => { const n = { ...prev }; delete n[fid]; return n })
      await loadFieldDetail(fid)
    } catch (e) {
      setError(String(e))
    }
  }

  async function deleteWindow(fid: string, wid: string) {
    try {
      await availabilityApi.delete(fid, wid)
      await loadFieldDetail(fid)
    } catch (e) {
      setError(String(e))
    }
  }

  function getBlackoutForm(fid: string): BlackoutForm {
    return newBlackout[fid] || { date: '', reason: '' }
  }

  function updateBlackoutForm(fid: string, updates: Partial<BlackoutForm>) {
    setNewBlackout(prev => ({ ...prev, [fid]: { ...getBlackoutForm(fid), ...updates } }))
  }

  async function createBlackout(e: React.FormEvent, fid: string) {
    e.preventDefault()
    const form = getBlackoutForm(fid)
    try {
      await blackoutApi.create(fid, {
        blackout_date: form.date,
        reason: form.reason || undefined,
      })
      setNewBlackout(prev => { const n = { ...prev }; delete n[fid]; return n })
      await loadFieldDetail(fid)
    } catch (e) {
      setError(String(e))
    }
  }

  async function deleteBlackout(fid: string, bid: string) {
    try {
      await blackoutApi.delete(fid, bid)
      await loadFieldDetail(fid)
    } catch (e) {
      setError(String(e))
    }
  }

  const inputStyle: React.CSSProperties = { padding: '0.4rem', borderRadius: 4, border: '1px solid #ccc' }
  const sectionHeadingStyle: React.CSSProperties = { marginTop: '1.25rem', marginBottom: '0.5rem', fontSize: '1rem', color: '#1a5276' }

  if (loading) return <div className="card"><p>Loading...</p></div>

  return (
    <div>
      <h1>Fields</h1>
      {error && (
        <div className="card" style={{ background: '#fdd' }}>
          <strong>Error:</strong> {error}
          <button
            onClick={() => setError(null)}
            style={{ marginLeft: '1rem', cursor: 'pointer', background: 'none', border: 'none', fontWeight: 'bold' }}
          >
            ✕
          </button>
        </div>
      )}

      <div className="card">
        <h2>Add Field</h2>
        <form onSubmit={createField} style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <label>
            Name<br />
            <input
              value={newField.name}
              onChange={e => setNewField(p => ({ ...p, name: e.target.value }))}
              required
              placeholder="e.g. Riverside Park Field 1"
              style={{ ...inputStyle, minWidth: 200 }}
            />
          </label>
          <label>
            Address<br />
            <input
              value={newField.address}
              onChange={e => setNewField(p => ({ ...p, address: e.target.value }))}
              placeholder="123 Main St"
              style={{ ...inputStyle, minWidth: 180 }}
            />
          </label>
          <label>
            Max Games/Day{' '}
            <span
              title="This is a cap on total games per day — it does not create slots. To allow 2 games on the same day, add 2 availability windows with different start times (e.g. 10:00 and 14:00)."
              style={{ cursor: 'help', color: '#888', fontSize: '0.85rem' }}
            >ⓘ</span><br />
            <input
              type="number"
              value={newField.maxGamesPerDay}
              min={1}
              onChange={e => setNewField(p => ({ ...p, maxGamesPerDay: Number(e.target.value) }))}
              style={{ ...inputStyle, width: 90 }}
            />
          </label>
          <button type="submit" className="btn btn-primary">Add Field</button>
        </form>
      </div>

      {fields.length === 0 && (
        <div className="card">
          <p className="placeholder">No fields yet. Add one above.</p>
        </div>
      )}

      {fields.map(field => (
        <div key={field.id} className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.5rem' }}>
            {editingFieldId === field.id ? (
              <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
                <label style={{ fontSize: '0.9rem' }}>
                  Name<br />
                  <input value={editFieldForm.name} onChange={e => setEditFieldForm(p => ({ ...p, name: e.target.value }))} style={inputStyle} />
                </label>
                <label style={{ fontSize: '0.9rem' }}>
                  Address<br />
                  <input value={editFieldForm.address} onChange={e => setEditFieldForm(p => ({ ...p, address: e.target.value }))} placeholder="optional" style={{ ...inputStyle, minWidth: 160 }} />
                </label>
                <label style={{ fontSize: '0.9rem' }}>
                  Max Games/Day{' '}
                  <span
                    title="This is a cap on total games per day — it does not create slots. To allow 2 games on the same day, add 2 availability windows with different start times (e.g. 10:00 and 14:00)."
                    style={{ cursor: 'help', color: '#888', fontSize: '0.85rem' }}
                  >ⓘ</span><br />
                  <input type="number" value={editFieldForm.maxGamesPerDay} min={1} onChange={e => setEditFieldForm(p => ({ ...p, maxGamesPerDay: Number(e.target.value) }))} style={{ ...inputStyle, width: 80 }} />
                </label>
                <div style={{ display: 'flex', gap: '0.25rem', alignSelf: 'flex-end' }}>
                  <button onClick={() => saveField(field.id)} className="btn btn-primary" style={{ fontSize: '0.85rem' }}>Save</button>
                  <button onClick={() => setEditingFieldId(null)} className="btn" style={{ fontSize: '0.85rem' }}>Cancel</button>
                </div>
              </div>
            ) : (
              <div>
                <strong style={{ fontSize: '1.05rem' }}>{field.name}</strong>
                {field.address && (
                  <span style={{ color: '#666', marginLeft: '0.75rem', fontSize: '0.9rem' }}>{field.address}</span>
                )}
                <span style={{ marginLeft: '1rem', color: '#888', fontSize: '0.85rem' }}>
                  Max {field.max_games_per_day} games/day
                </span>
                {!field.is_active && (
                  <span style={{ marginLeft: '0.75rem', background: '#eee', padding: '1px 8px', borderRadius: 10, fontSize: '0.8rem', color: '#666' }}>
                    Inactive
                  </span>
                )}
              </div>
            )}
            <div style={{ display: 'flex', gap: '0.5rem' }}>
              {editingFieldId !== field.id && (
                <button onClick={() => startEditField(field)} className="btn btn-primary" style={{ fontSize: '0.85rem' }}>Edit</button>
              )}
              <button onClick={() => toggleExpand(field.id)} className="btn btn-primary" style={{ fontSize: '0.85rem' }}>
                {expanded === field.id ? 'Collapse' : 'Manage Availability'}
              </button>
              <button onClick={() => deleteField(field.id)} className="btn btn-danger" style={{ fontSize: '0.85rem' }}>
                Delete
              </button>
            </div>
          </div>

          {expanded === field.id && (
            <div style={{ marginTop: '1rem', borderTop: '1px solid #eee', paddingTop: '1rem' }}>
              {/* Availability Windows */}
              <h3 style={sectionHeadingStyle}>Availability Windows</h3>
              <p style={{ fontSize: '0.83rem', color: '#555', background: '#f0f4f8', padding: '0.5rem 0.75rem', borderRadius: 4, margin: '0 0 0.75rem' }}>
                Each window defines <strong>one game slot</strong>. To allow multiple games on the same day,
                add multiple windows with different start times — e.g. one at 10:00 and another at 14:00 to
                allow two games. <em>Max Games/Day</em> is a safety cap, not a slot multiplier.
              </p>

              {(windows[field.id] || []).length === 0 && (
                <p style={{ color: '#888', fontSize: '0.9rem', fontStyle: 'italic' }}>No availability windows set.</p>
              )}

              {(windows[field.id] || []).map(w => (
                <div
                  key={w.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '0.4rem 0',
                    borderBottom: '1px solid #f0f0f0',
                    fontSize: '0.9rem',
                  }}
                >
                  <span>
                    <strong>{w.window_type}</strong>
                    {w.window_type === 'recurring' && w.days_of_week.length > 0 && (
                      <span style={{ marginLeft: '0.5rem', color: '#555' }}>
                        [{w.days_of_week.map(d => DAYS[d]).join(', ')}]
                      </span>
                    )}
                    <span style={{ marginLeft: '0.5rem' }}>
                      {w.start_date} &rarr; {w.end_date}
                    </span>
                    <span style={{ marginLeft: '0.5rem', color: '#555' }}>
                      {w.start_time} &ndash; {w.end_time}
                    </span>
                  </span>
                  <button
                    onClick={() => deleteWindow(field.id, w.id)}
                    className="btn btn-danger"
                    style={{ fontSize: '0.8rem', padding: '2px 10px' }}
                  >
                    Delete
                  </button>
                </div>
              ))}

              <form
                onSubmit={e => createWindow(e, field.id)}
                style={{ marginTop: '0.75rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end', background: '#f9f9f9', padding: '0.75rem', borderRadius: 6 }}
              >
                <label>
                  Type<br />
                  <select
                    value={getWindowForm(field.id).windowType}
                    onChange={e => updateWindowForm(field.id, { windowType: e.target.value })}
                    style={inputStyle}
                  >
                    <option value="recurring">Recurring</option>
                    <option value="oneoff">One-off</option>
                  </select>
                </label>

                {getWindowForm(field.id).windowType === 'recurring' && (
                  <label>
                    Days of Week<br />
                    <div style={{ display: 'flex', gap: 6, marginTop: 4, flexWrap: 'wrap' }}>
                      {DAYS.map((d, i) => (
                        <label key={i} style={{ fontSize: '0.8rem', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 2 }}>
                          <input
                            type="checkbox"
                            checked={getWindowForm(field.id).daysOfWeek.includes(i)}
                            onChange={() => toggleDay(field.id, i)}
                          />
                          {d}
                        </label>
                      ))}
                    </div>
                  </label>
                )}

                <label>
                  Start Date<br />
                  <input
                    type="date"
                    value={getWindowForm(field.id).startDate}
                    onChange={e => updateWindowForm(field.id, { startDate: e.target.value })}
                    required
                    style={inputStyle}
                  />
                </label>
                <label>
                  End Date<br />
                  <input
                    type="date"
                    value={getWindowForm(field.id).endDate}
                    onChange={e => updateWindowForm(field.id, { endDate: e.target.value })}
                    required
                    style={inputStyle}
                  />
                </label>
                <label>
                  Start Time<br />
                  <input
                    type="time"
                    value={getWindowForm(field.id).startTime}
                    onChange={e => updateWindowForm(field.id, { startTime: e.target.value })}
                    required
                    style={inputStyle}
                  />
                </label>
                <label>
                  End Time<br />
                  <input
                    type="time"
                    value={getWindowForm(field.id).endTime}
                    onChange={e => updateWindowForm(field.id, { endTime: e.target.value })}
                    required
                    style={inputStyle}
                  />
                </label>
                <button type="submit" className="btn btn-primary">Add Window</button>
              </form>

              {/* Blackout Dates */}
              <h3 style={{ ...sectionHeadingStyle, marginTop: '1.75rem' }}>Blackout Dates</h3>

              {(blackouts[field.id] || []).length === 0 && (
                <p style={{ color: '#888', fontSize: '0.9rem', fontStyle: 'italic' }}>No blackout dates set.</p>
              )}

              {(blackouts[field.id] || []).map(b => (
                <div
                  key={b.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '0.4rem 0',
                    borderBottom: '1px solid #f0f0f0',
                    fontSize: '0.9rem',
                  }}
                >
                  <span>
                    <strong>{b.blackout_date}</strong>
                    {b.reason && <span style={{ color: '#666', marginLeft: '0.5rem' }}> &mdash; {b.reason}</span>}
                  </span>
                  <button
                    onClick={() => deleteBlackout(field.id, b.id)}
                    className="btn btn-danger"
                    style={{ fontSize: '0.8rem', padding: '2px 10px' }}
                  >
                    Delete
                  </button>
                </div>
              ))}

              <form
                onSubmit={e => createBlackout(e, field.id)}
                style={{ marginTop: '0.75rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end', background: '#f9f9f9', padding: '0.75rem', borderRadius: 6 }}
              >
                <label>
                  Date<br />
                  <input
                    type="date"
                    value={getBlackoutForm(field.id).date}
                    onChange={e => updateBlackoutForm(field.id, { date: e.target.value })}
                    required
                    style={inputStyle}
                  />
                </label>
                <label>
                  Reason (optional)<br />
                  <input
                    value={getBlackoutForm(field.id).reason}
                    onChange={e => updateBlackoutForm(field.id, { reason: e.target.value })}
                    placeholder="e.g. Holiday"
                    style={{ ...inputStyle, minWidth: 160 }}
                  />
                </label>
                <button type="submit" className="btn btn-primary">Add Blackout</button>
              </form>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
