import { useState, useEffect } from 'react'
import { Settings2, Plus, Trash2, FolderOpen, FileCode, ChevronDown, ChevronRight } from 'lucide-react'
import { api, type TargetDir } from '../api/client'

export default function TargetsPage() {
  const [targets, setTargets] = useState<TargetDir[]>([])
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [newPath, setNewPath] = useState('')
  const [newLabel, setNewLabel] = useState('')
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [targetSkills, setTargetSkills] = useState<Record<number, string[]>>({})
  const [targetErrors, setTargetErrors] = useState<Record<number, string>>({})
  const [skillsLoading, setSkillsLoading] = useState<Record<number, boolean>>({})

  const loadTargets = async () => {
    setLoading(true)
    try {
      const data = await api.getTargets()
      setTargets(data.targets)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadTargets()
  }, [])

  const toggleExpand = async (id: number) => {
    if (expandedId === id) {
      setExpandedId(null)
      return
    }
    setExpandedId(id)

    // Load skills if not already loaded
    if (!targetSkills[id]) {
      setSkillsLoading((prev) => ({ ...prev, [id]: true }))
      setTargetErrors((prev) => ({ ...prev, [id]: '' }))
      try {
        const data = await api.getTargetSkills(id)
        setTargetSkills((prev) => ({ ...prev, [id]: data.skills }))
        if (data.error) {
          setTargetErrors((prev) => ({ ...prev, [id]: data.error! }))
        }
      } catch {
        setTargetSkills((prev) => ({ ...prev, [id]: [] }))
        setTargetErrors((prev) => ({ ...prev, [id]: '读取目录失败' }))
      } finally {
        setSkillsLoading((prev) => ({ ...prev, [id]: false }))
      }
    }
  }

  const handleAdd = async () => {
    if (!newPath.trim()) return
    try {
      await api.createTarget(newPath.trim(), newLabel.trim())
      setShowAdd(false)
      setNewPath('')
      setNewLabel('')
      loadTargets()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '添加失败'
      alert(msg)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('确定移除此目标目录？')) return
    try {
      await api.deleteTarget(id)
      if (expandedId === id) setExpandedId(null)
      loadTargets()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '删除失败'
      alert(msg)
    }
  }

  return (
    <div>
      <div className="flex items-end justify-between mb-6">
        <div>
          <h1 className="page-title">目标目录</h1>
          <p className="page-subtitle">Agent 启动时读取技能的位置</p>
        </div>
        <button className="btn-primary" onClick={() => setShowAdd(true)}>
          <Plus className="w-4 h-4" />
          添加目录
        </button>
      </div>

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="card-base p-4 animate-pulse">
              <div className="h-4 bg-border rounded w-1/2 mb-2" />
              <div className="h-3 bg-border rounded w-2/3" />
            </div>
          ))}
        </div>
      ) : targets.length === 0 ? (
        <div className="text-center py-16">
          <Settings2 className="w-12 h-12 text-border mx-auto mb-3" />
          <p className="text-sm text-fg-muted">
            还没有目标目录，点击右上角「添加目录」配置
          </p>
          <p className="text-xs text-fg-subtle mt-1">
            例如: ~/.claude/skills、~/.reasonix/skills
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {targets.map((t) => {
            const isExpanded = expandedId === t.id
            const skills = targetSkills[t.id]
            const skillCount = skills?.length ?? 0
            return (
              <div key={t.id} className="card-base overflow-hidden group">
                {/* Header */}
                <div
                  className="p-4 flex items-center gap-4 cursor-pointer select-none hover:bg-sidebar-hover/50 transition-colors"
                  onClick={() => toggleExpand(t.id)}
                >
                  <button className="p-0.5 rounded hover:bg-sidebar-hover transition-colors">
                    {isExpanded ? <ChevronDown className="w-4 h-4 text-fg-subtle" /> : <ChevronRight className="w-4 h-4 text-fg-subtle" />}
                  </button>
                  <div className="w-9 h-9 rounded-md bg-sidebar-bg flex items-center justify-center shrink-0">
                    <FolderOpen className="w-4 h-4 text-fg-muted" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{t.label || '未命名'}</p>
                    <p className="text-xs text-fg-muted font-mono truncate">{t.path}</p>
                  </div>
                  {skillCount > 0 && (
                    <span className="badge-accent text-xs">{skillCount} 个技能</span>
                  )}
                  <button
                    className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded hover:bg-sidebar-hover text-fg-muted hover:text-accent"
                    onClick={(e) => { e.stopPropagation(); handleDelete(t.id) }}
                    title="删除"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                {/* Expanded skills list */}
                {isExpanded && (
                  <div className="border-t border-border">
                    {skillsLoading[t.id] ? (
                      <div className="px-4 py-3 space-y-2">
                        {Array.from({ length: 3 }).map((_, i) => (
                          <div key={i} className="h-4 bg-border rounded w-1/3 animate-pulse" />
                        ))}
                      </div>
                    ) : targetErrors[t.id] ? (
                      <div className="px-4 py-6 text-center">
                        <p className="text-xs text-fg-muted">{targetErrors[t.id]}</p>
                      </div>
                    ) : !skills || skills.length === 0 ? (
                      <div className="px-4 py-6 text-center">
                        <FileCode className="w-6 h-6 text-border mx-auto mb-1.5" />
                        <p className="text-xs text-fg-muted">该目录暂无已同步的技能</p>
                      </div>
                    ) : (
                      <>
                        <div className="px-4 py-3 grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-1.5">
                          {skills.map((name) => (
                            <div key={name} className="flex items-center gap-2 px-2.5 py-1.5 rounded text-xs hover:bg-sidebar-hover transition-colors">
                              <FileCode className="w-3 h-3 text-accent shrink-0" />
                              <span className="truncate">{name}</span>
                            </div>
                          ))}
                        </div>
                        <div className="px-4 pb-3 flex justify-end">
                          <button
                            className="btn-ghost text-xs text-fg-muted hover:text-accent"
                            onClick={async () => {
                              if (!confirm(`确定清空此目录中所有已同步的技能？\n技能本身不会被删除，仅移除软链接。`)) return
                              try {
                                const res = await api.clearTarget(t.id)
                                setTargetSkills((prev) => {
                                  const next = { ...prev }
                                  delete next[t.id]
                                  return next
                                })
                                alert(`已清空 ${res.total} 个技能，备份至: ${res.backup_path}`)
                              } catch (err: unknown) {
                                const msg = err instanceof Error ? err.message : '清空失败'
                                alert(msg)
                              }
                            }}
                          >
                            清空
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Add Dialog */}
      {showAdd && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="card-base p-6 w-full max-w-md mx-4 shadow-dialog">
            <h2 className="text-lg font-display font-bold mb-4">添加目标目录</h2>

            <label className="text-sm font-medium text-fg-muted mb-1.5 block">目录路径</label>
            <input className="input-field mb-3" value={newPath} onChange={(e) => setNewPath(e.target.value)} placeholder="/home/user/.claude/skills" />

            <label className="text-sm font-medium text-fg-muted mb-1.5 block">标记（可选）</label>
            <input className="input-field mb-4" value={newLabel} onChange={(e) => setNewLabel(e.target.value)} placeholder="Claude 技能目录" />

            <div className="flex justify-end gap-2 pt-2 border-t border-border">
              <button className="btn-ghost" onClick={() => setShowAdd(false)}>取消</button>
              <button className="btn-primary" onClick={handleAdd}>添加</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
