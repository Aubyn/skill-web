import { BrowserRouter, Routes, Route } from 'react-router-dom'
import AppShell from './components/AppShell'
import SkillLibrary from './pages/SkillLibrary'
import SkillGroups from './pages/SkillGroups'
import SyncPage from './pages/SyncPage'
import TargetsPage from './pages/TargetsPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<SkillLibrary />} />
          <Route path="groups" element={<SkillGroups />} />
          <Route path="sync" element={<SyncPage />} />
          <Route path="targets" element={<TargetsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
