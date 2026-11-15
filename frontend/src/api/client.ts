export interface Skill {
  id: string
  name: string
  group: string
  source_path: string
  store_path: string
  skill_type: 'dir'
  description: string
  created_at: string
}

export interface SkillGroup {
  id: number
  name: string
  description: string
  skill_count: number
  created_at: string
}

export interface TargetDir {
  id: number
  path: string
  label: string
  skill_count: number
  created_at: string
}

export interface ScanResult {
  found: Array<{ id: string; name: string; skill_type: string; size: string; description: string }>
  existing: Record<string, string[]>
  suggested_group: string
  total: number
}

export interface ImportResult {
  group: string
  imported: Array<{ id: string; name: string; group: string; source: string; store_path: string }>
  conflicts: Array<{ name: string; action: string }>
  collection?: { id: number; name: string; created: boolean; skill_count: number }
  collection_error?: string
  total: number
}

export interface SyncRequest {
  skill_ids?: string[]
  group_ids?: number[]
  target_ids: number[]
  dry_run?: boolean
  copy_mode?: boolean
}

export interface SyncPreviewConflict {
  name: string
  winner: string
  skill_ids: string[]
}

export interface SyncPreviewTarget {
  target: string
  exists: boolean
  planned: string[]
  removed_old: string[]
  failed: string[]
}

export interface SyncPreviewResult {
  results: SyncPreviewTarget[]
  resolved_skills: string[]
  conflicts: SyncPreviewConflict[]
  total_planned: number
  missing_targets: string[]
}

export interface SyncResult {
  results: Array<{
    target: string
    synced: string[]
    failed: string[]
    removed_old: string[]
  }>
  resolved_skills: string[]
  backup_path: string
  total_synced: number
  mode?: string
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const err = await res.text()
    throw new Error(`API error ${res.status}: ${err}`)
  }
  return res.json()
}

export const api = {
  // Skills
  getSkills: (q = '', page = 1, pageSize = 50) =>
    request<{ skills: Skill[]; total: number; page: number; page_size: number }>(
      `/api/skills?q=${encodeURIComponent(q)}&page=${page}&page_size=${pageSize}`
    ),

  getSkill: (id: string) =>
    request<{ skill: Skill; files: string[] }>(`/api/skills/${encodeURIComponent(id)}`),

  deleteSkill: (id: string) =>
    request<{ success: boolean }>(`/api/skills/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  scanPreview: (dir: string) =>
    request<ScanResult>(`/api/scan-preview?dir=${encodeURIComponent(dir)}`),

  importSkills: (dir: string, group: string, createCollection: boolean, skillIds?: string[]) =>
    request<ImportResult>('/api/import', {
      method: 'POST',
      body: JSON.stringify({ dir, group, create_collection: createCollection, skill_ids: skillIds }),
    }),

  // Groups
  getGroups: (q = '') =>
    request<{ groups: SkillGroup[]; total: number }>(`/api/groups?q=${encodeURIComponent(q)}`),

  getGroup: (id: number) =>
    request<{ group: SkillGroup; skills: Skill[]; total: number }>(`/api/groups/${id}`),

  createGroup: (name: string, description: string, skillIds: string[]) =>
    request<{ group: SkillGroup }>('/api/groups', {
      method: 'POST',
      body: JSON.stringify({ name, description, skill_ids: skillIds }),
    }),

  deleteGroup: (id: number) =>
    request<{ success: boolean }>(`/api/groups/${id}`, { method: 'DELETE' }),

  addGroupSkills: (id: number, skillIds: string[]) =>
    request<{ group_id: number; added: number }>(`/api/groups/${id}/skills`, {
      method: 'POST',
      body: JSON.stringify({ skill_ids: skillIds }),
    }),

  removeGroupSkills: (id: number, skillIds: string[]) =>
    request<{ group_id: number; removed: number }>(`/api/groups/${id}/skills`, {
      method: 'DELETE',
      body: JSON.stringify({ skill_ids: skillIds }),
    }),

  // Targets
  getTargets: () =>
    request<{ targets: TargetDir[]; total: number }>('/api/targets'),

  createTarget: (path: string, label: string) =>
    request<{ target: TargetDir }>('/api/targets', {
      method: 'POST',
      body: JSON.stringify({ path, label }),
    }),

  getTargetSkills: (id: number) =>
    request<{ skills: string[]; total: number; error?: string }>(`/api/targets/${id}/skills`),

  clearTarget: (id: number) =>
    request<{ removed: string[]; failed: string[]; total: number; backup_path: string }>(`/api/targets/${id}/clear`, { method: 'POST' }),

  checkTargetsExist: (ids: number[]) =>
    request<{ results: Array<{ id: number; exists: boolean; path: string; error?: string }> }>(
      `/api/targets/exists-check?ids=${ids.join(',')}`
    ),

  deleteTarget: (id: number) =>
    request<{ success: boolean }>(`/api/targets/${id}`, { method: 'DELETE' }),

  // Sync
  sync: (req: SyncRequest) =>
    request<SyncResult>('/api/sync', {
      method: 'POST',
      body: JSON.stringify(req),
    }),

  syncPreview: (req: SyncRequest) =>
    request<SyncPreviewResult>('/api/sync', {
      method: 'POST',
      body: JSON.stringify({ ...req, dry_run: true }),
    }),
}
