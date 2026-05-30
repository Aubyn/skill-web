import { useState, useEffect } from 'react'
import { FolderSync, ArrowRight, CheckCircle2, Loader2 } from 'lucide-react'
import { api, type Skill, type SkillGroup, type TargetDir, type SyncResult } from '../api/client'

type SourceMode = 'skills' | 'groups'

interface SyncStatus {
  loading: boolean
  skillIds: string[]
  groupIds: number[]
  targetIds: number[]
  result: SyncResult | null
}

export default function SyncPage() {
  const [mode, setMode] = useState<SourceMode>('skills')
  const [skills, setSkills] = useState<Skill[]>([])
  const [groups, setGroups] = useState<SkillGroup[]>([])
  const [targets, setTargets] = useState<TargetDir[]>([])
  const [selectedSkills, setSelectedSkills] = useState<Set<string>>(new Set())
  const [selectedGroup, setSelectedGroup] = useState<number | null>(null)
  const [selectedTargets, setSelectedTargets] = useState<Set<number>>(new Set())
  const [sync, setSync] = useState<SyncStatus>({ loading: false, skillIds: [], groupIds: [], targetIds: [], result: null })

  useEffect(() => {
    api.getSkills('', 1, 500).then((d) => setSkills(d.skills)).catch(() => {})
    api.getGroups().then((d) => setGroups(d.groups)).catch(() => {})
    api.getTargets().then((d) => setTargets(d.targets)).catch(() => {})
  }, [])

  const toggleSkill = (id: string) => {
    setSelectedSkills((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleTarget = (id: number) => {
    setSelectedTargets((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleSync = async () => {
    if (sync.loading) return

    const skillIds = mode === 'skills' ? Array.from(selectedSkills) : []
    const groupIds = mode === 'groups' && selectedGroup !== null ? [selectedGroup] : []
    const targetIds = Array.from(selectedTargets)

    if (skillIds.length === 0 && groupIds.length === 0) return alert('请选择技能来源')
    if (targetIds.length === 0) return alert('请选择目标目录')

    if (mode === 'groups' && selectedGroup !== null) {
      const grp = groups.find((g) => g.id === selectedGroup)
      if (grp && grp.skill_count === 0) return alert('该技能组为空，无法同步')
    }

    if (!confirm('同步将覆盖目标目录中的原有内容，确定继续？')) return

    setSync({ loading: true, skillIds, groupIds, targetIds, result: null })
    try {
      const result = await api.sync({ skill_ids: skillIds, group_ids: groupIds, target_ids: targetIds })
      setSync({ loading: false, skillIds, groupIds, targetIds, result })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '同步失败'
      alert(msg)
      setSync({ loading: false, skillIds, groupIds, targetIds, result: null })
    }
  }

  const canSync = (mode === 'skills' && selectedSkills.size > 0) ||
    (mode === 'groups' && selectedGroup !== null) &&
    selectedTargets.size > 0

  return (
    <div>
      <div className="mb-6">
        <h1 className="page-title">同步</h1>
        <p className="page-subtitle">将技能部署到目标目录</p>
      </div>

      <div className="flex items-start gap-6">
        {/* Left: Source */}
        <div className="card-base p-5 flex-1 relative">
          {/* Arrow */}
          <div className="absolute right-0 top-1/2 -translate-y-1/2 translate-x-1/2 z-10 w-8 h-8 rounded-full bg-bg-card border border-border flex items-center justify-center">
            <ArrowRight className="w-4 h-4 text-fg-muted" />
          </div>

          <h2 className="text-sm font-semibold text-fg-base mb-3">来源</h2>

          {/* Mode Tabs */}
          <div className="flex gap-1 mb-4 bg-sidebar-bg rounded-md p-0.5">
            <button
              className={`flex-1 px-3 py-1.5 text-sm rounded font-medium transition-colors
                ${mode === 'skills' ? 'bg-white text-fg-base shadow-sm' : 'text-sidebar-text hover:text-fg-base'}`}
              onClick={() => setMode('skills')}
            >
              从技能库
            </button>
            <button
              className={`flex-1 px-3 py-1.5 text-sm rounded font-medium transition-colors
                ${mode === 'groups' ? 'bg-white text-fg-base shadow-sm' : 'text-sidebar-text hover:text-fg-base'}`}
              onClick={() => setMode('groups')}
            >
              从技能组
            </button>
          </div>

          {mode === 'skills' ? (
            <div className="space-y-1 max-h-80 overflow-y-auto">
              {skills.length === 0 ? (
                <p className="text-sm text-fg-muted text-center py-4">技能库为空</p>
              ) : (
                skills.map((s) => (
                  <label
                    key={s.id}
                    className={`flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors
                      ${selectedSkills.has(s.id) ? 'bg-accent-bg text-accent-dark' : 'hover:bg-sidebar-hover'}`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedSkills.has(s.id)}
                      onChange={() => toggleSkill(s.id)}
                      className="w-4 h-4 rounded border-border text-accent focus:ring-accent/30"
                    />
                    <span className="text-sm">{s.id}</span>
                  </label>
                ))
              )}
            </div>
          ) : (
            <div className="space-y-1">
              {groups.length === 0 ? (
                <p className="text-sm text-fg-muted text-center py-4">还没有技能组</p>
              ) : (
                groups.map((g) => (
                  <button
                    key={g.id}
                    onClick={() => setSelectedGroup(g.id)}
                    className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors
                      ${selectedGroup === g.id ? 'bg-accent-bg text-accent-dark font-medium' : 'hover:bg-sidebar-hover'}`}
                  >
                    <div className="flex items-center justify-between">
                      <span>{g.name}</span>
                      <span className="text-xs text-fg-subtle">{g.skill_count} 个技能</span>
                    </div>
                  </button>
                ))
              )}
            </div>
          )}
        </div>

        {/* Right: Target */}
        <div className="card-base p-5 flex-1">
          <h2 className="text-sm font-semibold text-fg-base mb-3">目标目录</h2>
          <div className="space-y-1">
            {targets.length === 0 ? (
              <p className="text-sm text-fg-muted text-center py-4">还没有目标目录</p>
            ) : (
              targets.map((t) => (
                <label
                  key={t.id}
                  className={`flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors
                    ${selectedTargets.has(t.id) ? 'bg-accent-bg text-accent-dark' : 'hover:bg-sidebar-hover'}`}
                >
                  <input
                    type="checkbox"
                    checked={selectedTargets.has(t.id)}
                    onChange={() => toggleTarget(t.id)}
                    className="w-4 h-4 rounded border-border text-accent focus:ring-accent/30"
                  />
                  <div className="min-w-0">
                    <span className="text-sm block truncate">{t.label || t.path}</span>
                    <span className="text-xs text-fg-subtle block truncate">{t.path}</span>
                  </div>
                </label>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Bottom: Sync Button */}
      <div className="mt-6 flex items-center justify-between">
        <div className="text-xs text-fg-muted">
          {mode === 'skills'
            ? `已选 ${selectedSkills.size} 个技能`
            : selectedGroup
              ? `已选技能组: ${groups.find((g) => g.id === selectedGroup)?.name}`
              : '未选择来源'}
          ，目标目录: {selectedTargets.size} 个
        </div>
        <button
          className="btn-primary"
          disabled={!canSync || sync.loading}
          onClick={handleSync}
        >
          {sync.loading ? (
            <><Loader2 className="w-4 h-4 animate-spin" /> 同步中…</>
          ) : (
            <><FolderSync className="w-4 h-4" /> 执行同步</>
          )}
        </button>
      </div>

      {/* Result */}
      {sync.result && (
        <div className="mt-4 card-base p-4 border-accent/30">
          <div className="flex items-center gap-2 mb-3">
            <CheckCircle2 className="w-5 h-5 text-green-600" />
            <span className="text-sm font-medium">
              同步完成 — 共 {sync.result.total_synced} 个技能
            </span>
            <span className="text-xs text-fg-muted ml-auto">
              备份: {sync.result.backup_path}
            </span>
          </div>
          {sync.result.results.map((r, i) => {
            const synced = r.synced || []
            const failed = r.failed || []
            const removed = r.removed_old || []
            return (
              <div key={i} className="mb-3 last:mb-0">
                <p className="text-xs font-semibold text-fg-base mb-1.5">{r.target}</p>
                {synced.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 mb-1.5">
                    {synced.map((name) => (
                      <span key={name} className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-green-50 text-green-700">
                        ✓ {name}
                      </span>
                    ))}
                  </div>
                )}
                {failed.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 mb-1.5">
                    {failed.map((name) => (
                      <span key={name} className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-red-50 text-red-600">
                        ✗ {name}
                      </span>
                    ))}
                  </div>
                )}
                {removed.length > 0 && (
                  <p className="text-xs text-fg-subtle">
                    已移除旧内容: {removed.join(', ')}
                  </p>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
