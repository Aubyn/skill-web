import { BrowserRouter, Routes, Route } from 'react-router-dom'
import AppShell from './components/AppShell'
import Dashboard from './pages/Dashboard'
import { ToastProvider } from './components/Toast'

export default function App() {
  return (
    <BrowserRouter>
      <ToastProvider>
        <Routes>
          <Route element={<AppShell />}>
            <Route index element={<Dashboard />} />
          </Route>
        </Routes>
      </ToastProvider>
    </BrowserRouter>
  )
}
