import { useState, useEffect } from 'react'
import { Layers, Plus, ChevronRight, FileCode, Search, X } from 'lucide-react'
import { api, type SkillGroup, type Skill } from '../api/client'

export default function SkillGroups() {
  const [groups, setGroups] = useState<SkillGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [selectedGroup, setSelectedGroup] = useState<number | null>(null)
  const [groupSkills, setGroupSkills] = useState<Skill[]>([])
  const [showAddSkills, setShowAddSkills] = useState(false)
  const [allSkills, setAllSkills] = useState<Skill[]>([])
  const [addSkillSearch, setAddSkillSearch] = useState('')
  const [selectedAddSkills, setSelectedAddSkills] = useState<Set<string>>(new Set())

  const loadGroups = async () => {
    setLoading(true)
    try {
      const data = await api.getGroups()
      setGroups(data.groups)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadGroups()
  }, [])

  const handleCreate = async () => {
    if (!newName.trim()) return
    try {
      await api.createGroup(newName.trim(), newDesc.trim(), [])
      setShowCreate(false)
      setNewName('')
      setNewDesc('')
      loadGroups()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '创建失败'
      alert(msg)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除此技能组？')) return
    try {
      await api.deleteGroup(id)
      if (selectedGroup === id) setSelectedGroup(null)
      loadGroups()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '删除失败'
      alert(msg)
    }
  }

  const openGroup = async (id: number) => {
    setSelectedGroup(id)
    const data = await api.getGroup(id)
    setGroupSkills(data.skills)
  }

  const openAddSkills = async () => {
    setShowAddSkills(true)
    setAddSkillSearch('')
    setSelectedAddSkills(new Set())
    const data = await api.getSkills('', 1, 100)
    setAllSkills(data.skills)
  }

  const searchAddSkills = async (term: string) => {
    setAddSkillSearch(term)
    const data = await api.getSkills(term, 1, 100)
    setAllSkills(data.skills)
  }

  return (
    <div className="flex gap-8 h-[calc(100vh-6rem)]">
      {/* Group List */}
      <div className="w-80 shrink-0 flex flex-col">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="page-title">技能组</h1>
            <p className="page-subtitle">共 {groups.length} 个组</p>
          </div>
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            <Plus className="w-4 h-4" />
            新建
          </button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-1">
          {loading ? (
            Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="card-base p-3 animate-pulse">
                <div className="h-4 bg-border rounded w-2/3" />
              </div>
            ))
          ) : groups.length === 0 ? (
            <div className="text-center py-8">
              <Layers className="w-8 h-8 text-border mx-auto mb-2" />
              <p className="text-xs text-fg-muted">还没有技能组</p>
            </div>
          ) : (
            groups.map((g) => (
              <button
                key={g.id}
                onClick={() => openGroup(g.id)}
                className={`w-full text-left card-base p-3 flex items-center gap-3
                  ${selectedGroup === g.id ? 'ring-2 ring-accent/30 border-accent/30' : ''}`}
              >
                <Layers className="w-4 h-4 text-accent shrink-0" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium truncate">{g.name}</p>
                  <p className="text-xs text-fg-subtle">{g.skill_count} 个技能</p>
                </div>
                <ChevronRight className="w-4 h-4 text-fg-subtle shrink-0" />
              </button>
            ))
          )}
        </div>
      </div>

      {/* Group Detail */}
      <div className="flex-1 border-l border-border pl-8">
        {selectedGroup ? (
          <>
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-display font-bold">
                  {groups.find((g) => g.id === selectedGroup)?.name}
                </h2>
                <p className="text-xs text-fg-muted">{groupSkills.length} 个技能</p>
              </div>
              <div className="flex gap-2">
                <button className="btn-ghost text-accent" onClick={openAddSkills}>
                  追加技能
                </button>
                <button className="btn-ghost text-accent" onClick={() => handleDelete(selectedGroup)}>
                  删除组
                </button>
              </div>
            </div>

            <div className="space-y-1">
              {groupSkills.length === 0 ? (
                <p className="text-sm text-fg-muted py-8 text-center">组内暂无技能</p>
              ) : (
                groupSkills.map((s) => (
                  <div key={s.id} className="card-base p-3 flex items-center gap-3 group">
                    <FileCode className="w-4 h-4 text-fg-subtle" />
                    <span className="text-sm flex-1">{s.id}</span>
                    <button
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-xs text-fg-muted hover:text-accent"
                      onClick={async () => {
                        await api.removeGroupSkills(selectedGroup, [s.id])
                        loadGroups()
                        openGroup(selectedGroup)
                      }}
                    >
                      移除
                    </button>
                  </div>
                ))
              )}
            </div>
          </>
        ) : (
          <div className="text-center py-16">
            <Layers className="w-10 h-10 text-border mx-auto mb-3" />
            <p className="text-sm text-fg-muted">选择一个技能组查看详情</p>
          </div>
        )}
      </div>

      {/* Create Dialog */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="card-base p-6 w-full max-w-sm mx-4 shadow-dialog">
            <h2 className="text-lg font-display font-bold mb-4">新建技能组</h2>
            <label className="text-sm font-medium text-fg-muted mb-1.5 block">组名</label>
            <input className="input-field mb-3" value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="core-dev" />
            <label className="text-sm font-medium text-fg-muted mb-1.5 block">描述（可选）</label>
            <input className="input-field mb-4" value={newDesc} onChange={(e) => setNewDesc(e.target.value)} placeholder="核心开发技能集合" />
            <div className="flex justify-end gap-2 pt-2 border-t border-border">
              <button className="btn-ghost" onClick={() => setShowCreate(false)}>取消</button>
              <button className="btn-primary" onClick={handleCreate}>创建</button>
            </div>
          </div>
        </div>
      )}

      {/* Add Skills Dialog */}
      {showAddSkills && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="card-base p-6 w-full max-w-lg mx-4 shadow-dialog">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-display font-bold">追加技能</h2>
              <button className="btn-ghost p-1" onClick={() => setShowAddSkills(false)}>
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Search */}
            <div className="relative mb-3">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-fg-subtle" />
              <input
                className="input-field pl-9 text-sm"
                placeholder="搜索技能…"
                value={addSkillSearch}
                onChange={(e) => searchAddSkills(e.target.value)}
              />
            </div>

            {/* Skill list */}
            <div className="max-h-64 overflow-y-auto space-y-0.5 mb-4">
              {allSkills
                .filter((s) => !groupSkills.find((gs) => gs.id === s.id))
                .filter((s) => !addSkillSearch || s.id.toLowerCase().includes(addSkillSearch.toLowerCase()))
                .map((s) => (
                  <label
                    key={s.id}
                    className={`flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors
                      ${selectedAddSkills.has(s.id) ? 'bg-accent-bg text-accent-dark' : 'hover:bg-sidebar-hover'}`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedAddSkills.has(s.id)}
                      onChange={() => {
                        setSelectedAddSkills((prev) => {
                          const next = new Set(prev)
                          if (next.has(s.id)) next.delete(s.id)
                          else next.add(s.id)
                          return next
                        })
                      }}
                      className="w-4 h-4 rounded border-border text-accent focus:ring-accent/30"
                    />
                    <FileCode className="w-4 h-4 text-fg-subtle shrink-0" />
                    <span className="text-sm">{s.id}</span>
                  </label>
                ))}
              {allSkills.filter((s) => !groupSkills.find((gs) => gs.id === s.id)).length === 0 && (
                <p className="text-sm text-fg-muted text-center py-6">所有技能已在组中</p>
              )}
            </div>

            <div className="flex items-center justify-between pt-2 border-t border-border">
              <span className="text-xs text-fg-muted">已选 {selectedAddSkills.size} 个技能</span>
              <div className="flex gap-2">
                <button className="btn-ghost" onClick={() => setShowAddSkills(false)}>取消</button>
                <button
                  className="btn-primary"
                  disabled={selectedAddSkills.size === 0}
                  onClick={async () => {
                    await api.addGroupSkills(selectedGroup!, Array.from(selectedAddSkills))
                    setShowAddSkills(false)
                    loadGroups()
                    openGroup(selectedGroup!)
                  }}
                >
                  确认追加
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
