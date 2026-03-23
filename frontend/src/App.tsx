import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom'
import TeamsPage from './pages/TeamsPage'
import FieldsPage from './pages/FieldsPage'
import SeasonsPage from './pages/SeasonsPage'
import SchedulePage from './pages/SchedulePage'

export default function App() {
  return (
    <BrowserRouter>
      <nav>
        <NavLink to="/teams">Teams</NavLink>
        <NavLink to="/fields">Fields</NavLink>
        <NavLink to="/seasons">Seasons</NavLink>
        <NavLink to="/schedule">Schedule</NavLink>
      </nav>
      <main>
        <Routes>
          <Route path="/" element={<TeamsPage />} />
          <Route path="/teams" element={<TeamsPage />} />
          <Route path="/fields" element={<FieldsPage />} />
          <Route path="/seasons" element={<SeasonsPage />} />
          <Route path="/schedule" element={<SchedulePage />} />
        </Routes>
      </main>
    </BrowserRouter>
  )
}
