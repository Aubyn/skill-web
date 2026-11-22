import type { SyncPreviewResult, SyncResult, SyncStrategy } from '../api/client'

/**
 * Whether a sync plan is destructive enough to need an explicit confirmation.
 *
 * Conflicts always confirm (a target can only hold one skill per name).
 * Append confirms when an existing same-name skill is about to be replaced;
 * apply confirms when existing target content is about to be removed.
 */
export function needsConfirmation(preview: SyncPreviewResult, strategy: SyncStrategy): boolean {
  if (preview.conflicts.length > 0) return true
  return preview.results.some(r =>
    strategy === 'append' ? r.replaced.length > 0 : r.removed_old.length > 0
  )
}

export interface SyncSummary {
  total: number
  failed: number
  details: string
}

/** Folds a sync result into the text shown by the result toast. */
export function summarizeSync(result: SyncResult): SyncSummary {
  const failed = result.results.reduce((sum, r) => sum + r.failed.length, 0)
  const details = result.results
    .map(r => `${r.target}: ${r.synced.join(', ')}`)
    .join(' | ')
  return { total: result.total_synced, failed, details }
}

/** Short button label for a strategy. */
export function strategyLabel(strategy: SyncStrategy): string {
  return strategy === 'append' ? '追加' : '应用'
}

/** One-line explanation shown in the sync bar. */
export function strategyHint(strategy: SyncStrategy): string {
  return strategy === 'append'
    ? '追加：保留目标目录中的其它技能，只覆盖同名技能'
    : '应用：备份并清空目标目录后全量部署'
}
