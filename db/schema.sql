-- Skill library
CREATE TABLE IF NOT EXISTS skills (
    id          TEXT PRIMARY KEY,
    source_path TEXT NOT NULL,
    store_path  TEXT NOT NULL,
    skill_type  TEXT NOT NULL DEFAULT 'dir' CHECK(skill_type = 'dir'),
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Skill groups
CREATE TABLE IF NOT EXISTS skill_groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Group ↔ skill mapping
CREATE TABLE IF NOT EXISTS group_skills (
    group_id INTEGER NOT NULL REFERENCES skill_groups(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, skill_id)
);

-- Target directories
CREATE TABLE IF NOT EXISTS target_dirs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT NOT NULL UNIQUE,
    label       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
