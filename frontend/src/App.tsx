import { Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { BoardPage } from './pages/BoardPage'
import { ChangeDetailPage } from './pages/ChangeDetailPage'
import './App.css'

export default function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<BoardPage />} />
        <Route path="/projects/:projectID/changes/:name" element={<ChangeDetailPage />} />
      </Routes>
    </AppShell>
  )
}
