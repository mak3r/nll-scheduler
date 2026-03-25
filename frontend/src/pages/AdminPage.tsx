import { useState } from 'react'
import { adminApi, type FullBundle } from '../api/admin'

type Status = 'idle' | 'loading' | 'success' | 'error'

export default function AdminPage() {
  const [exportStatus, setExportStatus] = useState<Status>('idle')
  const [importStatus, setImportStatus] = useState<Status>('idle')
  const [error, setError] = useState<string | null>(null)
  const [importSuccess, setImportSuccess] = useState<string | null>(null)

  async function handleExport() {
    setExportStatus('loading')
    setError(null)
    try {
      const bundle = await adminApi.exportAll()
      const blob = new Blob([JSON.stringify(bundle, null, 2)], {
        type: 'application/json',
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `nll-export-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
      setExportStatus('success')
    } catch (e) {
      setError(String(e))
      setExportStatus('error')
    }
  }

  async function handleImport(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setImportStatus('loading')
    setError(null)
    setImportSuccess(null)
    try {
      const text = await file.text()
      const bundle: FullBundle = JSON.parse(text)
      if (bundle.version !== '1') {
        throw new Error(`Unknown bundle version: ${bundle.version}`)
      }
      const divisionNames = bundle.teams.divisions.map(d => d.name).join(', ')
      const teamNames = bundle.teams.teams.map(t => t.name).join(', ')
      const fieldNames = bundle.fields.fields.map(f => f.name).join(', ')
      const seasonNames = bundle.schedule.seasons.map(s => s.name).join(', ')
      if (
        !confirm(
          `Import bundle from ${bundle.exported_at}?\n\n` +
            `This will upsert:\n` +
            `  Divisions (${bundle.teams.divisions.length}): ${divisionNames}\n` +
            `  Teams (${bundle.teams.teams.length}): ${teamNames}\n` +
            `  Fields (${bundle.fields.fields.length}): ${fieldNames}\n` +
            `  Seasons (${bundle.schedule.seasons.length}): ${seasonNames}\n\n` +
            `Existing records with the same ID will be overwritten.`,
        )
      ) {
        setImportStatus('idle')
        e.target.value = ''
        return
      }
      await adminApi.importAll(bundle)
      setImportSuccess(
        `Import complete:\n` +
          `  Divisions: ${divisionNames}\n` +
          `  Fields: ${fieldNames}\n` +
          `  Seasons: ${seasonNames}`,
      )
      setImportStatus('success')
    } catch (err) {
      setError(String(err))
      setImportStatus('error')
    } finally {
      e.target.value = ''
    }
  }

  return (
    <div>
      <h1>Admin</h1>

      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      <section>
        <h2>Export</h2>
        <p>Download a full bundle of all divisions, teams, fields, and schedules as JSON.</p>
        <button onClick={handleExport} disabled={exportStatus === 'loading'}>
          {exportStatus === 'loading' ? 'Exporting…' : 'Export All Data'}
        </button>
        {exportStatus === 'success' && <span> Download started.</span>}
      </section>

      <section>
        <h2>Import</h2>
        <p>
          Restore from a previously exported bundle. Existing records with the same ID will be
          updated; records with new IDs will be created.
        </p>
        <input
          type="file"
          accept=".json,application/json"
          onChange={handleImport}
          disabled={importStatus === 'loading'}
        />
        {importStatus === 'loading' && <p>Importing…</p>}
        {importSuccess && <p style={{ color: 'green' }}>{importSuccess}</p>}
      </section>
    </div>
  )
}
