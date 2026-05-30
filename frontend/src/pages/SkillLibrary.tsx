import { useState, useEffect, useCallback } from 'react'
import { Search, Plus, FolderOpen, Trash2, FileCode } from 'lucide-react'
import { api, type Skill } from '../api/client'

export default function SkillLibrary() {
  const [skills, setSkills] = useState<Skill[]>([])
  const [total, setTotal] = useState(0)
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [showImport, setShowImport] = useState(false)
  const [importDir, setImportDir] = useState('')
  const [importPreview, setImportPreview] = useState<import('../api/client').ScanResult | null>(null)
  const [importing, setImporting] = useState(false)

  const loadSkills = useCallback(async (q: string) => {
    setLoading(true)
    try {
      const data = await api.getSkills(q)
      setSkills(data.skills)
      setTotal(data.total)
    } catch {
      // silently fail — show empty
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadSkills(query)
  }, [loadSkills, query])

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
      loadSkills(query)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '导入失败'
      alert(msg)
    } finally {
      setImporting(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(`确定要从技能库中删除「${id}」？`)) return
    try {
      await api.deleteSkill(id)
      loadSkills(query)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '删除失败'
      alert(msg)
    }
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-end justify-between mb-6">
        <div>
          <h1 className="page-title">技能库</h1>
          <p className="page-subtitle">
            {loading ? '加载中…' : `共 ${total} 个技能`}
          </p>
        </div>
        <button className="btn-primary" onClick={() => setShowImport(true)}>
          <Plus className="w-4 h-4" />
          添加技能
        </button>
      </div>

      {/* Search */}
      <div className="relative mb-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-fg-subtle" />
        <input
          className="input-field pl-10"
          placeholder="搜索技能…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      {/* Import Dialog */}
      {showImport && (
        <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="card-base p-6 w-full max-w-lg mx-4 shadow-dialog">
            <h2 className="text-lg font-display font-bold mb-4">导入技能</h2>

            <label className="text-sm font-medium text-fg-muted mb-1.5 block">
              本地目录路径
            </label>
            <div className="flex gap-2 mb-4">
              <input
                className="input-field flex-1"
                placeholder="/home/user/.claude/skills"
                value={importDir}
                onChange={(e) => setImportDir(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleScan()}
              />
              <button className="btn-outline" onClick={handleScan}>
                <FolderOpen className="w-4 h-4" />
                扫描
              </button>
            </div>

            {/* Preview */}
            {importPreview && (
              <div className="mb-4">
                <p className="text-sm font-medium text-fg-base mb-2">
                  发现 {importPreview.total} 个技能
                  {(importPreview.conflicts_with_existing || []).length > 0 && (
                    <span className="text-accent ml-2">
                      （{(importPreview.conflicts_with_existing || []).length} 个同名冲突，将覆盖）
                    </span>
                  )}
                </p>
                <div className="max-h-40 overflow-y-auto space-y-1">
                  {importPreview.found.map((s) => (
                    <div key={s.id} className="flex items-center gap-2 px-2 py-1 rounded text-sm hover:bg-sidebar-hover">
                      <FileCode className="w-3.5 h-3.5 text-fg-subtle" />
                      <span>{s.id}</span>
                      <span className="text-xs text-fg-subtle ml-auto">{s.size}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2 border-t border-border">
              <button className="btn-ghost" onClick={() => { setShowImport(false); setImportPreview(null) }}>
                取消
              </button>
              <button
                className="btn-primary"
                disabled={!importPreview || importing}
                onClick={handleImport}
              >
                {importing ? '导入中…' : '确认导入'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Skill Grid */}
      {loading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="card-base p-4 animate-pulse">
              <div className="h-4 bg-border rounded w-2/3 mb-2" />
              <div className="h-3 bg-border rounded w-1/2" />
            </div>
          ))}
        </div>
      ) : skills.length === 0 ? (
        <div className="text-center py-16">
          <FileCode className="w-12 h-12 text-border mx-auto mb-3" />
          <p className="text-fg-muted text-sm">
            {query ? '没有匹配的技能' : '还没有技能，点击右上角「添加技能」开始'}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {skills.map((skill) => (
            <div key={skill.id} className="card-hover p-4 group">
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3 min-w-0">
                  <div className="w-8 h-8 rounded-md bg-accent-bg flex items-center justify-center shrink-0 mt-0.5">
                    <FileCode className="w-4 h-4 text-accent" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="font-medium text-fg-base truncate">
                      {skill.id}
                    </h3>
                    <p className="text-xs text-fg-subtle mt-0.5 truncate">
                      {skill.store_path}
                    </p>
                  </div>
                </div>
                <button
                  className="opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-sidebar-hover text-fg-muted hover:text-accent"
                  onClick={() => handleDelete(skill.id)}
                  title="删除"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
