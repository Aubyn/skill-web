import { Outlet } from 'react-router-dom'

export default function AppShell() {
  return (
    <div className="h-screen bg-bg-base overflow-hidden">
      <main className="h-full overflow-hidden">
        <Outlet />
      </main>
    </div>
  )
}
