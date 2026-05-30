import { useState, useEffect, useCallback } from 'react'
import {
  Search, Plus, FolderOpen, Trash2, FileCode, Layers, ChevronRight,
  ChevronDown, Settings2, FolderOpen as FolderIcon, Loader2,
  ArrowRightToLine, Eye, Check
} from 'lucide-react'
import { api, type Skill, type SkillGroup, type TargetDir, type ScanResult } from '../api/client'
import { ScrollArea } from '../components/ScrollArea'
import { useToast } from '../components/Toast'

type SourceMode = 'skills' | 'groups'

export default function Dashboard() {
  // ── Data ──
  const [skills, setSkills] = useState<Skill[]>([])
  const [groups, setGroups] = useState<SkillGroup[]>([])
  const [targets, setTargets] = useState<TargetDir[]>([])
  const [groupSkills, setGroupSkills] = useState<Record<number, Skill[]>>({})
  const [targetSkills, setTargetSkills] = useState<Record<number, string[]>>({})
  const [targetErrors, setTargetErrors] = useState<Record<number, string>>({})
  const [loading, setLoading] = useState(true)

  // ── Left Column: Source ──
  const [sourceMode, setSourceMode] = useState<SourceMode>('skills')
  const [query, setQuery] = useState('')
  const [selectedSkills, setSelectedSkills] = useState<Set<string>>(new Set())
  const [expandedGroup, setExpandedGroup] = useState<number | null>(null)

  // ── Right Column: Target ──
  const [selectedTargets, setSelectedTargets] = useState<Set<number>>(new Set())
  const [expandedTarget, setExpandedTarget] = useState<number | null>(null)

  // ── Welcome ──
  const [showWelcome, setShowWelcome] = useState<'first' | 'reopen' | null>(() =>
    localStorage.getItem('skill-web-welcome-dismissed') ? null : 'first'
  )

  const dismissWelcome = () => {
    localStorage.setItem('skill-web-welcome-dismissed', '1')
    setShowWelcome(null)
  }

  // ── Sync ──
  const [syncing, setSyncing] = useState(false)
  const { toast } = useToast()

  // ── Dialogs ──
  const [showImport, setShowImport] = useState(false)
  const [importDir, setImportDir] = useState('')
  const [importPreview, setImportPreview] = useState<ScanResult | null>(null)
  const [importing, setImporting] = useState(false)

  const [showCreateGroup, setShowCreateGroup] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [newGroupDesc, setNewGroupDesc] = useState('')

  const [showAddSkills, setShowAddSkills] = useState(false)
  const [addSkillSearch, setAddSkillSearch] = useState('')
  const [selectedAddSkills, setSelectedAddSkills] = useState<Set<string>>(new Set())
  const [allSkills, setAllSkills] = useState<Skill[]>([])

  const [showAddTarget, setShowAddTarget] = useState(false)
  const [newTargetPath, setNewTargetPath] = useState('')
  const [newTargetLabel, setNewTargetLabel] = useState('')

  // ── Load data ──
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [sd, gd, td] = await Promise.all([
        api.getSkills(''),
        api.getGroups(),
        api.getTargets(),
      ])
      setSkills(sd.skills)
      setGroups(gd.groups)
      setTargets(td.targets)
    } catch { /* ignore */ }
    setLoading(false)
  }, [])

  useEffect(() => { loadData() }, [loadData])

  // ── Search (only in skills mode) ──
  const handleSearch = useCallback(async (q: string) => {
    setQuery(q)
    try {
      const data = await api.getSkills(q)
      setSkills(data.skills)
    } catch { /* ignore */ }
  }, [])

  // ── Left: source selection ──
  const toggleSkill = (id: string) => {
    setSelectedSkills(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  const toggleTarget = (id: number) => {
    setSelectedTargets(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  const toggleExpandGroup = async (id: number) => {
    if (expandedGroup === id) { setExpandedGroup(null); return }
    setExpandedGroup(id)
    if (!groupSkills[id]) {
      try {
        const data = await api.getGroup(id)
        setGroupSkills(prev => ({ ...prev, [id]: data.skills }))
      } catch { setGroupSkills(prev => ({ ...prev, [id]: [] })) }
    }
  }

  const toggleExpandTarget = async (id: number) => {
    if (expandedTarget === id) { setExpandedTarget(null); return }
    setExpandedTarget(id)
    if (!targetSkills[id]) {
      try {
        const data = await api.getTargetSkills(id)
        setTargetSkills(prev => ({ ...prev, [id]: data.skills }))
        if (data.error) setTargetErrors(prev => ({ ...prev, [id]: data.error! }))
      } catch { setTargetSkills(prev => ({ ...prev, [id]: [] })) }
    }
  }

  // ── Left: delete skill ──
  const handleDeleteSkill = async (id: string) => {
    if (!confirm(`确定要从技能库中删除「${id}」？`)) return
    try {
      await api.deleteSkill(id)
      if (query) handleSearch(query); else loadData()
    } catch { alert('删除失败') }
  }

  // ── Left: remove skill from group ──
  const handleRemoveGroupSkill = async (groupId: number, skillId: string) => {
    try {
      await api.removeGroupSkills(groupId, [skillId])
      const data = await api.getGroup(groupId)
      setGroupSkills(prev => ({ ...prev, [groupId]: data.skills }))
      loadData()
    } catch { alert('移除失败') }
  }

  // ── Left: delete group ──
  const handleDeleteGroup = async (id: number) => {
    if (!confirm('确定删除此技能组？')) return
    try {
      await api.deleteGroup(id)
      if (expandedGroup === id) setExpandedGroup(null)
      loadData()
    } catch { alert('删除失败') }
  }

  // ── Right: clear target ──
  const handleClearTarget = async (id: number) => {
    if (!confirm(`确定清空此目录中所有已同步的技能？\n技能本身不会被删除，仅移除软链接。`)) return
    try {
      await api.clearTarget(id)
      setTargetSkills(prev => { const n = { ...prev }; delete n[id]; return n })
      setExpandedTarget(null)
    } catch { alert('清空失败') }
  }

  // ── Right: delete target ──
  const handleDeleteTarget = async (id: number) => {
    if (!confirm('确定移除此目标目录？')) return
    try {
      await api.deleteTarget(id)
      if (expandedTarget === id) { setExpandedTarget(null); setTargetSkills(prev => { const n = { ...prev }; delete n[id]; return n }) }
      loadData()
    } catch { alert('删除失败') }
  }

  // ── Sync ──
  const handleSync = async () => {
    if (syncing) return
    const targetIds = Array.from(selectedTargets)
    if (targetIds.length === 0) return alert('请在右侧勾选目标目录')

    let skillIds: string[] = []
    let groupIds: number[] = []

    if (sourceMode === 'skills') {
      skillIds = Array.from(selectedSkills)
      if (skillIds.length === 0) return alert('请在左侧勾选要同步的技能')
    } else {
      if (expandedGroup === null) return alert('请在左侧展开一个技能组')
      groupIds = [expandedGroup]
    }

    // Check if target directories exist
    const existCheck = await api.checkTargetsExist(targetIds)
    const missing = existCheck.results.filter(r => !r.exists)
    if (missing.length > 0) {
      const paths = missing.map(r => r.path).join('\n')
      if (!confirm(`以下目标目录不存在，是否自动创建？\n\n${paths}`)) return
    }

    if (!confirm('同步将覆盖目标目录中的原有内容，确定继续？')) return

    setSyncing(true)
    try {
      const result = await api.sync({ skill_ids: skillIds, group_ids: groupIds, target_ids: targetIds })
      const total = result.total_synced
      const failed = result.results.reduce((n: number, r: any) => n + (r.failed?.length || 0), 0)
      const details = result.results.map((r: any) => `${r.target}: ${(r.synced||[]).join(', ')}`).join(' | ')
      toast({
        type: failed > 0 ? 'error' : 'success',
        title: failed > 0 ? `同步完成，${failed} 个失败` : `同步成功 — ${total} 个技能`,
        message: details,
      })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '同步失败'
      toast({ type: 'error', title: '同步失败', message: msg })
    }
    setSyncing(false)
  }

  // ── Skill Detail ──
  const [skillDetail, setSkillDetail] = useState<{ skill: Skill; files: string[] } | null>(null)

  const loadSkillDetail = async (id: string) => {
    try {
      const data = await api.getSkill(id)
      setSkillDetail(data)
    } catch {
      toast({ type: 'error', title: '加载技能详情失败' })
    }
  }

  // ── Import ──
  const handleScan = async () => {
    if (!importDir.trim()) return
    try {
      const result = await api.scanPreview(importDir)
      setImportPreview(result)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '扫描失败'
      alert(msg)
    }
  }

  const handleImport = async () => {
    if (!importPreview) return
    setImporting(true)
    try {
      await api.importSkills(importDir)
      setShowImport(false)
      setImportDir('')
      setImportPreview(null)
      loadData()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '导入失败'
      alert(msg)
    }
    setImporting(false)
  }

  // ── Create Group ──
  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return
    try {
      // If skills tab has selected skills, pre-populate the group
      const skillIds = sourceMode === 'skills' ? Array.from(selectedSkills) : []
      await api.createGroup(newGroupName.trim(), newGroupDesc.trim(), skillIds)
      setShowCreateGroup(false)
      setNewGroupName('')
      setNewGroupDesc('')
      if (skillIds.length > 0) setSelectedSkills(new Set())
      loadData()
    } catch { alert('创建失败') }
  }

  // ── Add Skills to Group ──
  const openAddSkills = async () => {
    setSelectedAddSkills(new Set())
    setAddSkillSearch('')
    const data = await api.getSkills('', 1, 500)
    setAllSkills(data.skills)
    setShowAddSkills(true)
  }

  const searchAddSkills = async (term: string) => {
    setAddSkillSearch(term)
    const data = await api.getSkills(term, 1, 500)
    setAllSkills(data.skills)
  }

  const handleAddSkills = async () => {
    if (!showAddSkills || !expandedGroup || selectedAddSkills.size === 0) return
    try {
      await api.addGroupSkills(expandedGroup, Array.from(selectedAddSkills))
      setShowAddSkills(false)
      const data = await api.getGroup(expandedGroup)
      setGroupSkills(prev => ({ ...prev, [expandedGroup]: data.skills }))
      loadData()
    } catch { alert('追加失败') }
  }

  // ── Add Target ──
  const handleAddTarget = async () => {
    if (!newTargetPath.trim()) return
    try {
      await api.createTarget(newTargetPath.trim(), newTargetLabel.trim())
      setShowAddTarget(false)
      setNewTargetPath('')
      setNewTargetLabel('')
      loadData()
    } catch { alert('添加失败') }
  }

  // ── Filtered skills ──
  const filteredSkills = query ? skills : skills

  return (
    <div className="h-full flex flex-col">
      {/* ── Top Bar ── */}
      <div className="shrink-0 px-8 pt-5 pb-4 border-b border-border/40">
        <div className="flex items-center gap-2">
          <h1 className="text-base font-bold text-fg-base tracking-tight">技能管理器</h1>
          <button onClick={() => setShowWelcome('reopen')} className="p-1 rounded hover:bg-sidebar-hover text-fg-subtle hover:text-fg-base transition-colors" title="重新显示引导">
            <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><path d="M12 17h.01"/>
            </svg>
          </button>
        </div>
      </div>

      {/* ── Main: 2 Columns ── */}
      <div className="flex-1 flex gap-0 min-h-0">
        {/* ════════ LEFT COLUMN ════════ */}
        <div className="flex-1 flex flex-col min-w-0 border-r border-border/40 bg-bg-muted/40">
          {/* Tabs */}
          <div className="shrink-0 flex items-center gap-1 px-6 pt-4 pb-3">
            <button
              onClick={() => { setSourceMode('skills'); if (query) handleSearch(query) }}
              className={`px-3 py-1.5 text-sm rounded-md font-medium transition-all
                ${sourceMode === 'skills' ? 'bg-bg-card text-fg-base shadow-sm' : 'text-fg-muted hover:text-fg-base'}`}
            >
              全部技能
            </button>
            <button
              onClick={() => setSourceMode('groups')}
              className={`px-3 py-1.5 text-sm rounded-md font-medium transition-all
                ${sourceMode === 'groups' ? 'bg-bg-card text-fg-base shadow-sm' : 'text-fg-muted hover:text-fg-base'}`}
            >
              技能组
            </button>
            {sourceMode === 'skills' && (
              <>
                <span className="ml-auto" />
                {selectedSkills.size > 0 && (
                  <button className="btn-ghost text-xs" onClick={() => {
                    setNewGroupName('')
                    setNewGroupDesc('')
                    setShowCreateGroup(true)
                  }}>
                    <Layers className="w-3.5 h-3.5" /> 新建组
                  </button>
                )}
                <button className="btn-ghost text-xs" onClick={() => setShowImport(true)}>
                  <Plus className="w-3.5 h-3.5" /> 导入
                </button>
              </>
            )}
            {sourceMode === 'groups' && (
              <button className="btn-ghost text-xs ml-auto" onClick={() => setShowCreateGroup(true)}>
                <Layers className="w-3.5 h-3.5" /> 新建组
              </button>
            )}
          </div>

          {/* Search (only in skills mode) */}
          {sourceMode === 'skills' && (
            <div className="shrink-0 flex items-center gap-2 px-6 pb-3">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-fg-subtle pointer-events-none" />
                <input
                  className="input-field pl-9 text-sm"
                  placeholder="搜索技能…"
                  value={query}
                  onChange={e => handleSearch(e.target.value)}
                />
              </div>
            </div>
          )}

          {/* Scrollable list */}
          <ScrollArea className="flex-1 px-6 pb-4">
            {loading ? (
              <div className="space-y-2 pt-2">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="h-10 bg-border/40 rounded-lg animate-pulse" />
                ))}
              </div>
            ) : sourceMode === 'skills' ? (
              filteredSkills.length === 0 ? (
                <div className="text-center pt-8 pb-2">
                  <FileCode className="w-8 h-8 text-border mx-auto mb-2" />
                  <p className="text-sm font-medium text-fg-muted mb-1">还没有技能</p>
                  <p className="text-xs text-fg-subtle mb-3">技能是含 SKILL.md 的目录，导入后统一管理</p>
                  <button className="btn-outline text-xs" onClick={() => setShowImport(true)}>
                    <FolderOpen className="w-3.5 h-3.5" /> 导入技能
                  </button>
                </div>
              ) : (
                <div className="space-y-0.5">
                  {filteredSkills.map(s => (
                    <div
                      key={s.id}
                      onClick={() => toggleSkill(s.id)}
                      className={`flex items-center gap-3 px-3.5 py-2.5 rounded-lg cursor-pointer transition-all group
                        ${selectedSkills.has(s.id)
                          ? 'bg-accent-bg/15'
                          : 'hover:bg-sidebar-hover'}`}
                    >
                      <div className={`w-4 h-4 rounded-[4px] flex items-center justify-center shrink-0 mt-0.5 transition-all cursor-pointer
                        ${selectedSkills.has(s.id)
                          ? 'bg-accent text-white'
                          : 'border border-border bg-white'}`}
                      >
                        {selectedSkills.has(s.id) && <Check className="w-3 h-3 stroke-[3]" />}
                      </div>
                      <FileCode className="w-4 h-4 text-fg-subtle shrink-0 mt-1" />
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium truncate">{s.id}</div>
                        {s.description && (
                          <div className="text-xs text-fg-muted truncate" title={s.description}>{s.description}</div>
                        )}
                      </div>
                      <button
                        onClick={e => { e.stopPropagation(); loadSkillDetail(s.id) }}
                        className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-bg-base text-fg-muted hover:text-fg-base transition-opacity"
                        title="查看详情"
                      >
                        <Eye className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={e => { e.stopPropagation(); handleDeleteSkill(s.id) }}
                        className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-bg-base text-fg-muted hover:text-accent transition-opacity"
                        title="删除"
                      >
                        <Trash2 className="w-3 h-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )
            ) : (
              /* Group mode */
              groups.length === 0 ? (
                <div className="text-center pt-12">
                  <Layers className="w-10 h-10 text-border mx-auto mb-2" />
                  <p className="text-sm text-fg-muted">还没有技能组</p>
                </div>
              ) : (
                <div className="space-y-0.5">
                  {groups.map(g => (
                    <div key={g.id} className="rounded-lg overflow-hidden">
                      {/* Group header */}
                      <div
                        onClick={() => toggleExpandGroup(g.id)}
                        className={`flex items-center gap-3 px-3.5 py-2.5 rounded-lg cursor-pointer transition-all group
                          ${expandedGroup === g.id
                            ? 'bg-bg-card shadow-sm border border-border/60'
                            : 'border border-transparent hover:bg-sidebar-hover'}`}
                      >
                        {expandedGroup === g.id
                          ? <ChevronDown className="w-4 h-4 text-fg-subtle shrink-0" />
                          : <ChevronRight className="w-4 h-4 text-fg-subtle shrink-0" />
                        }
                        <Layers className="w-4 h-4 text-accent shrink-0" />
                        <span className="text-sm font-medium flex-1 min-w-0 truncate">{g.name}</span>
                        <span className="text-xs text-fg-muted tabular-nums">{g.skill_count}</span>
                        <button
                          onClick={e => {
                            e.stopPropagation()
                            if (expandedGroup !== g.id) toggleExpandGroup(g.id)
                            openAddSkills()
                          }}
                          className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-bg-base text-fg-muted hover:text-accent transition-opacity"
                          title="追加技能"
                        >
                          <Plus className="w-3 h-3" />
                        </button>
                        <button
                          onClick={e => { e.stopPropagation(); handleDeleteGroup(g.id) }}
                          className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-bg-base text-fg-muted hover:text-accent transition-opacity"
                          title="删除组"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>

                      {/* Group children */}
                      {expandedGroup === g.id && (
                        <div className="ml-4 mt-1 mb-1 pl-4 border-l-2 border-border/40 space-y-0.5">
                          {groupSkills[g.id]?.length === 0 || !groupSkills[g.id] ? (
                            <p className="text-xs text-fg-muted py-3 text-center">组内暂无技能</p>
                          ) : (
                            (groupSkills[g.id] || []).map(sk => (
                              <div key={sk.id} className="flex items-center gap-3 px-3 py-2 rounded-lg group/child hover:bg-sidebar-hover transition-colors">
                                <FileCode className="w-3.5 h-3.5 text-fg-subtle shrink-0 mt-0.5" />
                                <div className="flex-1 min-w-0">
                                  <div className="text-sm truncate">{sk.id}</div>
                                  {sk.description && (
                                    <div className="text-xs text-fg-muted truncate" title={sk.description}>{sk.description}</div>
                                  )}
                                </div>
                                <button
                                  onClick={() => handleRemoveGroupSkill(g.id, sk.id)}
                                  className="opacity-0 group-hover/child:opacity-100 text-xs text-fg-muted hover:text-accent transition-opacity shrink-0"
                                >
                                  移除
                                </button>
                              </div>
                            ))
                          )}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )
            )}
          </ScrollArea>
        </div>

        {/* ════════ RIGHT COLUMN ════════ */}
        <div className="w-[420px] shrink-0 flex flex-col min-h-0">
          {/* Header */}
          <div className="shrink-0 flex items-center justify-between px-6 pt-4 pb-3">
            <div className="flex items-center gap-2">
              <Settings2 className="w-4 h-4 text-accent" />
              <h2 className="text-sm font-semibold text-fg-base">目标目录</h2>
            </div>
            <button className="btn-ghost text-xs" onClick={() => setShowAddTarget(true)}>
              <Plus className="w-3.5 h-3.5" /> 添加
            </button>
          </div>

          {/* Target list */}
          <ScrollArea className="flex-1 px-6 pb-4 space-y-2">
            {loading ? (
              Array.from({ length: 2 }).map((_, i) => (
                <div key={i} className="h-16 bg-border/40 rounded-xl animate-pulse" />
              ))
            ) : targets.length === 0 ? (
              <div className="text-center pt-8 pb-2">
                <Settings2 className="w-8 h-8 text-border mx-auto mb-2" />
                <p className="text-sm font-medium text-fg-muted mb-1">还没有目标目录</p>
                <p className="text-xs text-fg-subtle mb-3">目标目录是 AI 读取技能的位置，如 ~/.claude/skills</p>
                <button className="btn-outline text-xs" onClick={() => setShowAddTarget(true)}>
                  <Plus className="w-3.5 h-3.5" /> 添加目录
                </button>
              </div>
            ) : (
              targets.map(t => (
                <div key={t.id} className="card-base overflow-hidden group">
                  {/* Target header */}
                  <div
                    onClick={() => toggleExpandTarget(t.id)}
                    className={`flex items-center gap-3 px-4 py-3 cursor-pointer select-none transition-colors
                      ${expandedTarget === t.id ? '' : 'hover:bg-sidebar-hover/50'}`}
                  >
                    <div onClick={e => { e.stopPropagation(); toggleTarget(t.id) }} className={`w-4 h-4 rounded-[4px] flex items-center justify-center shrink-0 transition-all cursor-pointer
                        ${selectedTargets.has(t.id)
                          ? 'bg-accent text-white'
                          : 'border border-border bg-white'}`}
                    >
                      {selectedTargets.has(t.id) && <Check className="w-3 h-3 stroke-[3]" />}
                    </div>
                    <FolderIcon className="w-4 h-4 text-accent shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{t.label || '未命名'}</p>
                      <p className="text-xs text-fg-muted font-mono truncate">{t.path}</p>
                    </div>
                    {(targetSkills[t.id]?.length ?? 0) > 0 && (
                      <span className="badge-accent text-xs">{targetSkills[t.id].length}</span>
                    )}
                    <button
                      onClick={e => { e.stopPropagation(); handleDeleteTarget(t.id) }}
                      className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-bg-base text-fg-muted hover:text-accent transition-opacity"
                      title="删除"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  {/* Expanded skills */}
                  {expandedTarget === t.id && (
                    <div className="border-t border-border/40">
                      {targetSkills[t.id] === undefined ? (
                        <div className="px-4 py-3 space-y-2">
                          {Array.from({ length: 2 }).map((_, i) => (
                            <div key={i} className="h-4 bg-border/40 rounded w-1/3 animate-pulse" />
                          ))}
                        </div>
                      ) : targetErrors[t.id] ? (
                        <div className="px-4 py-4 text-center">
                          <p className="text-xs text-fg-muted">{targetErrors[t.id]}</p>
                        </div>
                      ) : !targetSkills[t.id] || targetSkills[t.id].length === 0 ? (
                        <div className="px-4 py-4 text-center">
                          <p className="text-xs text-fg-muted">该目录暂无已同步的技能</p>
                        </div>
                      ) : (
                        <>
                          <div className="px-4 py-3 flex flex-wrap gap-1.5">
                            {(targetSkills[t.id] || []).map(name => (
                              <span key={name} className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs bg-accent-bg text-accent-dark">
                                <FileCode className="w-3 h-3" />
                                {name}
                              </span>
                            ))}
                          </div>
                          <div className="px-4 pb-3 flex justify-end">
                            <button
                              className="btn-ghost text-xs text-fg-muted hover:text-accent"
                              onClick={() => handleClearTarget(t.id)}
                            >
                              清空
                            </button>
                          </div>
                        </>
                      )}
                    </div>
                  )}
                </div>
              ))
            )}
          </ScrollArea>

          {/* ── Sync Bar ── */}
          <div className="shrink-0 border-t border-border/40 px-6 py-4 space-y-3 bg-sidebar-bg/40">
            <div className="flex items-center justify-between">
              <div className="text-xs text-fg-muted">
                {sourceMode === 'skills' ? (
                  <>已选 <span className="font-medium text-fg-base tabular-nums">{selectedSkills.size}</span> 个技能</>
                ) : expandedGroup ? (
                  <>已选组: <span className="font-medium text-fg-base">{groups.find(g => g.id === expandedGroup)?.name || '未知'}</span></>
                ) : (
                  <>未选择来源</>
                )}
                → <span className="font-medium text-fg-base tabular-nums">{selectedTargets.size}</span> 个目标
              </div>
              <button
                className="btn-primary text-sm"
                disabled={syncing || selectedTargets.size === 0 || (sourceMode === 'skills' ? selectedSkills.size === 0 : expandedGroup === null)}
                onClick={handleSync}
              >
                {syncing ? (
                  <><Loader2 className="w-4 h-4 animate-spin" /> 同步中…</>
                ) : (
                  <><ArrowRightToLine className="w-4 h-4" /> 执行同步</>
                )}
              </button>
            </div>
            <div className="bg-bg-base/60 rounded-lg px-3 py-2">
              <p className="text-xs text-fg-muted flex items-start gap-1.5">
                <span className="text-base shrink-0">&#9888;&#65039;</span>
                <span>同步会覆盖目标目录所有内容，同步前自动备份到 ~/.skill-web/backups/</span>
              </p>
            </div>


          </div>
        </div>
      </div>

      {/* ════════════════════════════════════ */}
      {/* ── DIALOGS ── */}
      {/* ════════════════════════════════════ */}

      {/* Import Dialog */}
      {showImport && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-bg-card rounded-xl p-6 w-full max-w-lg mx-4 shadow-dialog border border-border/60">
            <h2 className="text-base font-semibold mb-4">导入技能</h2>





<label className="text-sm font-medium text-fg-muted mb-1.5 block">目录路径</label>
            <div className="flex gap-2 mb-4">
              <input className="input-field flex-1" placeholder="/home/user/我的技能目录"
                value={importDir} onChange={e => { setImportDir(e.target.value); setImportPreview(null) }}
                onKeyDown={e => e.key === 'Enter' && handleScan()} />
              <button className="btn-outline" onClick={handleScan}><FolderOpen className="w-4 h-4" /> 扫描</button>
            </div>

            {/* Scan results */}
            {importPreview ? (
              <div className="mb-4 px-4 py-3 rounded-xl bg-accent-bg/30 border border-accent/20">
                <p className="text-sm font-medium text-fg-base mb-2">
                  ✓ 扫描完成 — 发现 {importPreview.total} 个技能
                  {(importPreview.conflicts_with_existing || []).length > 0 && (
                    <span className="text-accent ml-2">（{(importPreview.conflicts_with_existing || []).length} 个已存在，将覆盖）</span>
                  )}
                </p>
                <div className="max-h-40 overflow-y-auto space-y-0.5">
                  {importPreview.found.map(s => (
                    <div key={s.id} className="flex items-center gap-2 px-2 py-1 rounded text-sm hover:bg-bg-card">
                      <FileCode className="w-3.5 h-3.5 text-fg-subtle shrink-0" />
                      <span>{s.id}</span>
                      <span className="text-xs text-fg-muted ml-auto">{s.size}</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : importDir.trim() ? (
              <div className="mb-4 px-4 py-3 rounded-xl bg-border/20 border border-dashed border-border/60">
                <p className="text-xs text-fg-muted flex items-center gap-1.5">
                  <span className="text-base">&#128203;</span>
                  先点击「扫描」预览技能。技能是含 SKILL.md 的目录。
                </p>
              </div>
            ) : null}

            {/* Actions */}
            <div className="flex justify-end gap-2 pt-2 border-t border-border/60">
              <button className="btn-ghost" onClick={() => { setShowImport(false); setImportPreview(null); setImportDir('') }}>取消</button>
              <button className="btn-primary" disabled={!importPreview || importing} onClick={handleImport}>
                {importing ? (
                  <><Loader2 className="w-4 h-4 animate-spin" /> 导入中…</>
                ) : importPreview ? (
                  <>确认导入 {importPreview.total} 个技能</>
                ) : (
                  <>请先扫描</>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Group Dialog */}
      {showCreateGroup && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-bg-card rounded-xl p-6 w-full max-w-md mx-4 shadow-dialog border border-border/60">
            <h2 className="text-base font-semibold mb-4">新建技能组</h2>
            <label className="text-sm font-medium text-fg-muted mb-1.5 block">组名</label>
            <input className="input-field mb-3" value={newGroupName} onChange={e => setNewGroupName(e.target.value)} placeholder="core-dev" />
            <label className="text-sm font-medium text-fg-muted mb-1.5 block">描述（可选）</label>
            <input className="input-field mb-4" value={newGroupDesc} onChange={e => setNewGroupDesc(e.target.value)} placeholder="核心开发技能集合" />
            <div className="flex justify-end gap-2 pt-2 border-t border-border/60">
              <button className="btn-ghost" onClick={() => { setShowCreateGroup(false); setNewGroupName(''); setNewGroupDesc('') }}>取消</button>
              <button className="btn-primary" onClick={handleCreateGroup}>创建</button>
            </div>
          </div>
        </div>
      )}

      {/* Add Skills to Group Dialog */}
      {showAddSkills && expandedGroup !== null && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-bg-card rounded-xl p-6 w-full max-w-md mx-4 shadow-dialog border border-border/60">
            <h2 className="text-base font-semibold mb-4">追加技能到组</h2>
            <div className="relative mb-3">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-fg-subtle" />
              <input className="input-field pl-9 text-sm" placeholder="搜索技能…"
                value={addSkillSearch} onChange={e => searchAddSkills(e.target.value)} />
            </div>
            <ScrollArea className="max-h-64 mb-4 space-y-0.5">
              {allSkills
                .filter(s => !(groupSkills[expandedGroup] || []).find(gs => gs.id === s.id))
                .map(s => (
                  <label key={s.id}
                    className={`flex items-start gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors
                      ${selectedAddSkills.has(s.id) ? 'bg-accent-bg text-accent-dark' : 'hover:bg-sidebar-hover'}`}>
                    <div onClick={e => { e.stopPropagation(); setSelectedAddSkills(prev => { const next = new Set(prev); if (next.has(s.id)) next.delete(s.id); else next.add(s.id); return next }) }} className={`w-4 h-4 rounded-[4px] flex items-center justify-center shrink-0 mt-1 transition-all cursor-pointer
                        ${selectedAddSkills.has(s.id)
                          ? 'bg-accent text-white'
                          : 'border border-border bg-white'}`}
                    >
                      {selectedAddSkills.has(s.id) && <Check className="w-3 h-3 stroke-[3]" />}
                    </div>
                    <div className="min-w-0">
                      <div className="text-sm font-medium">{s.id}</div>
                      {s.description && (
                        <div className="text-xs text-fg-muted truncate" title={s.description}>{s.description}</div>
                      )}
                    </div>
                  </label>
                ))}
              {allSkills.filter(s => !(groupSkills[expandedGroup] || []).find(gs => gs.id === s.id)).length === 0 && (
                <p className="text-sm text-fg-muted text-center py-6">所有技能已在组中</p>
              )}
            </ScrollArea>
            <div className="flex items-center justify-between pt-2 border-t border-border/60">
              <span className="text-xs text-fg-muted">已选 {selectedAddSkills.size} 个技能</span>
              <div className="flex gap-2">
                <button className="btn-ghost" onClick={() => setShowAddSkills(false)}>取消</button>
                <button className="btn-primary" disabled={selectedAddSkills.size === 0} onClick={handleAddSkills}>确认追加</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Welcome Dialog */}
      {showWelcome && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-bg-card rounded-xl p-6 w-full max-w-md mx-4 shadow-dialog border border-border/60">
            <h2 className="text-base font-semibold flex items-center gap-2 mb-1">
              <Layers className="w-5 h-5 text-accent" />
              技能管理器
            </h2>
            <p className="text-sm text-fg-muted mb-5 leading-relaxed">
              你拥有技能，你决定 AI 用什么。<br/>
              AI 从目标目录加载技能。<br/>
              这里是你管理技能的地方。
            </p>

            <div className="space-y-2.5 mb-5">
              <div className="flex items-start gap-3 px-3.5 py-2.5 rounded-lg bg-accent-bg/20">
                <span className="w-5 h-5 rounded-full bg-accent text-white text-xs font-bold flex items-center justify-center shrink-0 mt-0.5">1</span>
                <div>
                  <p className="text-sm font-medium">导入已有技能到库</p>
                  <p className="text-xs text-fg-muted mt-0.5">扫描本地目录，收录含 SKILL.md 的技能</p>
                </div>
              </div>
              <div className="flex items-start gap-3 px-3.5 py-2.5 rounded-lg bg-accent-bg/20">
                <span className="w-5 h-5 rounded-full bg-accent text-white text-xs font-bold flex items-center justify-center shrink-0 mt-0.5">2</span>
                <div>
                  <p className="text-sm font-medium">按场景分组</p>
                  <p className="text-xs text-fg-muted mt-0.5">把技能组合成群，按需使用</p>
                </div>
              </div>
              <div className="flex items-start gap-3 px-3.5 py-2.5 rounded-lg bg-accent-bg/20">
                <span className="w-5 h-5 rounded-full bg-accent text-white text-xs font-bold flex items-center justify-center shrink-0 mt-0.5">3</span>
                <div>
                  <p className="text-sm font-medium">同步给 AI</p>
                  <p className="text-xs text-fg-muted mt-0.5">选择哪些技能给 AI，同步后 AI 那边跟着增减</p>
                </div>
              </div>
            </div>

            <div className="bg-bg-base rounded-lg px-3.5 py-2.5 mb-5">
              <p className="text-xs font-medium text-accent-dark flex items-center gap-1.5">
                <span className="text-base">&#9888;&#65039;</span>
                同步前自动备份旧内容<br/>
                选错了或后悔了，去 ~/.skill-web/backups/ 找回
              </p>
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-border/60">
              {showWelcome === 'first' ? (
                <button className="btn-primary" onClick={dismissWelcome}>知道了，开始使用</button>
              ) : (
                <button className="btn-ghost" onClick={() => setShowWelcome(null)}>关闭</button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Add Target Dialog */}
      {showAddTarget && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-bg-card rounded-xl p-6 w-full max-w-md mx-4 shadow-dialog border border-border/60">
            <h2 className="text-base font-semibold mb-4">添加目标目录</h2>
            <label className="text-sm font-medium text-fg-muted mb-1.5 block">目录路径</label>
            <input className="input-field mb-3" value={newTargetPath} onChange={e => setNewTargetPath(e.target.value)} placeholder="/home/user/.claude/skills" />
            <label className="text-sm font-medium text-fg-muted mb-1.5 block">标签（可选）</label>
            <input className="input-field mb-4" value={newTargetLabel} onChange={e => setNewTargetLabel(e.target.value)} placeholder="Claude 技能目录" />
            <div className="flex justify-end gap-2 pt-2 border-t border-border/60">
              <button className="btn-ghost" onClick={() => { setShowAddTarget(false); setNewTargetPath(''); setNewTargetLabel('') }}>取消</button>
              <button className="btn-primary" onClick={handleAddTarget}>添加</button>
            </div>
          </div>
        </div>
      )}

      {/* Skill Detail Dialog */}
      {skillDetail && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50" onClick={() => setSkillDetail(null)}>
          <div className="bg-bg-card rounded-xl p-6 w-full max-w-lg mx-4 shadow-dialog border border-border/60" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold flex items-center gap-2">
                <FileCode className="w-4 h-4 text-accent" />
                {skillDetail.skill.id}
              </h2>
              <button onClick={() => setSkillDetail(null)} className="p-1 rounded hover:bg-sidebar-hover text-fg-muted hover:text-fg-base transition-colors">
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
              </button>
            </div>

            {/* Metadata */}
            <div className="space-y-2 mb-4 text-sm">
              <div className="flex justify-between">
                <span className="text-fg-muted">类型</span>
                <span className="font-medium">{skillDetail.skill.skill_type}</span>
              </div>
              {skillDetail.skill.description && (
                <div className="flex flex-col gap-0.5">
                  <span className="text-fg-muted">描述</span>
                  <span className="text-sm">{skillDetail.skill.description}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-fg-muted">来源路径</span>
                <span className="text-xs font-mono text-right max-w-[70%] truncate" title={skillDetail.skill.source_path}>{skillDetail.skill.source_path}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-fg-muted">存储路径</span>
                <span className="text-xs font-mono text-right max-w-[70%] truncate" title={skillDetail.skill.store_path}>{skillDetail.skill.store_path}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-fg-muted">导入时间</span>
                <span className="text-xs">{skillDetail.skill.created_at}</span>
              </div>
            </div>

            {/* Files */}
            <div>
              <p className="text-sm font-medium text-fg-muted mb-2">文件结构</p>
              {skillDetail.files.length === 0 ? (
                <p className="text-xs text-fg-subtle">（空目录）</p>
              ) : (
                <div className="bg-bg-base rounded-lg p-3 max-h-48 overflow-y-auto space-y-0.5">
                  {skillDetail.files.map(f => (
                    <div key={f} className="text-xs font-mono text-fg-muted px-1 py-0.5 hover:text-fg-base">{f}</div>
                  ))}
                </div>
              )}
            </div>

            <div className="flex justify-end pt-2 mt-4 border-t border-border/60">
              <button className="btn-ghost text-sm" onClick={() => setSkillDetail(null)}>关闭</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
