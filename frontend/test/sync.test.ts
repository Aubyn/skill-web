import assert from 'node:assert/strict'
import { test } from 'node:test'

import { needsConfirmation, strategyHint, strategyLabel, summarizeSync } from '../src/lib/sync.ts'
import type { SyncPreviewResult, SyncResult } from '../src/api/client.ts'

function preview(overrides: Partial<SyncPreviewResult> = {}): SyncPreviewResult {
  return {
    results: [
      {
        target: '/t',
        exists: true,
        planned: ['x'],
        removed_old: [],
        replaced: [],
        kept: [],
        failed: [],
      },
    ],
    resolved_skills: ['x'],
    conflicts: [],
    total_planned: 1,
    missing_targets: [],
    ...overrides,
  }
}

test('apply confirms only when existing content would be removed', () => {
  assert.equal(needsConfirmation(preview(), 'apply'), false)
  assert.equal(
    needsConfirmation(
      preview({ results: [{ ...preview().results[0], removed_old: ['old'] }] }),
      'apply'
    ),
    true
  )
})

test('append confirms only when a same-name skill would be replaced', () => {
  assert.equal(needsConfirmation(preview(), 'append'), false)
  assert.equal(
    needsConfirmation(
      preview({ results: [{ ...preview().results[0], kept: ['other'] }] }),
      'append'
    ),
    false
  )
  assert.equal(
    needsConfirmation(
      preview({ results: [{ ...preview().results[0], replaced: ['x'] }] }),
      'append'
    ),
    true
  )
})

test('selection conflicts always confirm', () => {
  const withConflict = preview({
    conflicts: [{ name: 'x', winner: 'a:x', skill_ids: ['a:x', 'b:x'] }],
  })
  assert.equal(needsConfirmation(withConflict, 'append'), true)
  assert.equal(needsConfirmation(withConflict, 'apply'), true)
})

test('summarizeSync counts failures and lists synced skills per target', () => {
  const result: SyncResult = {
    results: [
      { target: '/a', synced: ['x', 'y'], failed: [], removed_old: [], replaced: [], kept: [] },
      { target: '/b', synced: ['z'], failed: ['q'], removed_old: [], replaced: [], kept: [] },
    ],
    resolved_skills: ['x', 'y', 'z', 'q'],
    backup_path: '/backups/1',
    total_synced: 3,
  }
  assert.deepEqual(summarizeSync(result), {
    total: 3,
    failed: 1,
    details: '/a: x, y | /b: z',
  })
})

test('strategy copy matches the button and dialog wording', () => {
  assert.equal(strategyLabel('append'), '追加')
  assert.equal(strategyLabel('apply'), '应用')
  assert.match(strategyHint('append'), /只覆盖同名技能/)
  assert.match(strategyHint('apply'), /清空目标目录/)
})
