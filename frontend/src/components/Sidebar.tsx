import { NavLink } from 'react-router-dom'
import {
  Package,
  Layers,
  FolderSync,
  Settings2,
} from 'lucide-react'

const navItems = [
  { to: '/', label: '技能库', icon: Package },
  { to: '/groups', label: '技能组', icon: Layers },
  { to: '/sync', label: '同步', icon: FolderSync },
  { to: '/targets', label: '目标目录', icon: Settings2 },
]

export default function Sidebar() {
  return (
    <aside className="w-sidebar h-screen bg-sidebar-bg border-r border-border flex flex-col shrink-0">
      {/* Logo */}
      <div className="px-5 pt-6 pb-4 border-b border-border/60">
        <h1 className="font-display text-lg font-bold text-fg-base tracking-tight">
          技能管理器
        </h1>
        <p className="text-xs text-fg-subtle mt-0.5">Skill Web</p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              `sidebar-link ${isActive ? 'active' : ''}`
            }
          >
            <item.icon className="w-4 h-4 shrink-0" />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="px-5 py-4 border-t border-border/60">
        <p className="text-xs text-fg-subtle">v0.1.0</p>
      </div>
    </aside>
  )
}
