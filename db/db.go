package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type DB struct {
	*sql.DB
}

func Open() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	dir := filepath.Join(home, ".skill-web")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	path := filepath.Join(dir, "skill-web.db")
	sqldb, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqldb.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes

	if _, err := sqldb.Exec(schema); err != nil {
		return nil, fmt.Errorf("exec schema: %w", err)
	}

	return &DB{sqldb}, nil
}

// ---- Skills ----

func (d *DB) ListSkills(q string, page, pageSize int) ([]Skill, int, error) {
	where := ""
	args := []any{}
	if q != "" {
		where = "WHERE id LIKE ?"
		args = append(args, "%"+q+"%")
	}

	var total int
	countQ := "SELECT COUNT(*) FROM skills " + where
	if err := d.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf("SELECT id, source_path, store_path, skill_type, created_at FROM skills %s ORDER BY id LIMIT ? OFFSET ?", where)
	args = append(args, pageSize, offset)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.SourcePath, &s.StorePath, &s.SkillType, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		skills = append(skills, s)
	}
	return skills, total, nil
}

func (d *DB) GetSkill(id string) (*Skill, error) {
	var s Skill
	err := d.QueryRow("SELECT id, source_path, store_path, skill_type, created_at FROM skills WHERE id = ?", id).
		Scan(&s.ID, &s.SourcePath, &s.StorePath, &s.SkillType, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) UpsertSkill(s *Skill) error {
	_, err := d.Exec(`INSERT INTO skills (id, source_path, store_path, skill_type) VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET source_path=excluded.source_path, store_path=excluded.store_path`,
		s.ID, s.SourcePath, s.StorePath, s.SkillType)
	return err
}

func (d *DB) DeleteSkill(id string) error {
	_, err := d.Exec("DELETE FROM skills WHERE id = ?", id)
	return err
}

func (d *DB) SkillExists(id string) (bool, error) {
	var n int
	err := d.QueryRow("SELECT COUNT(*) FROM skills WHERE id = ?", id).Scan(&n)
	return n > 0, err
}

// ---- Groups ----

func (d *DB) ListGroups(q string) ([]SkillGroup, error) {
	where := ""
	args := []any{}
	if q != "" {
		where = "WHERE sg.name LIKE ?"
		args = append(args, "%"+q+"%")
	}

	query := fmt.Sprintf(`
		SELECT sg.id, sg.name, sg.description, sg.created_at,
			(SELECT COUNT(*) FROM group_skills WHERE group_id = sg.id) as skill_count
		FROM skill_groups sg %s
		ORDER BY sg.created_at DESC`, where)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []SkillGroup
	for rows.Next() {
		var g SkillGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.SkillCount); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *DB) CreateGroup(name, desc string) (*SkillGroup, error) {
	res, err := d.Exec("INSERT INTO skill_groups (name, description) VALUES (?, ?)", name, desc)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &SkillGroup{ID: int(id), Name: name, Description: desc, SkillCount: 0}, nil
}

func (d *DB) GetGroup(id int) (*SkillGroup, error) {
	var g SkillGroup
	err := d.QueryRow(`
		SELECT sg.id, sg.name, sg.description, sg.created_at,
			(SELECT COUNT(*) FROM group_skills WHERE group_id = sg.id) as skill_count
		FROM skill_groups sg WHERE sg.id = ?`, id).
		Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.SkillCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &g, err
}

func (d *DB) DeleteGroup(id int) error {
	_, err := d.Exec("DELETE FROM skill_groups WHERE id = ?", id)
	return err
}

func (d *DB) GetGroupSkills(groupID int) ([]Skill, error) {
	rows, err := d.Query(`
		SELECT s.id, s.source_path, s.store_path, s.skill_type, s.created_at
		FROM skills s
		JOIN group_skills gs ON gs.skill_id = s.id
		WHERE gs.group_id = ?
		ORDER BY s.id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.SourcePath, &s.StorePath, &s.SkillType, &s.CreatedAt); err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, nil
}

func (d *DB) AddGroupSkills(groupID int, skillIDs []string) (int, error) {
	added := 0
	for _, sid := range skillIDs {
		_, err := d.Exec("INSERT OR IGNORE INTO group_skills (group_id, skill_id) VALUES (?, ?)", groupID, sid)
		if err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

func (d *DB) RemoveGroupSkills(groupID int, skillIDs []string) (int, error) {
	removed := 0
	for _, sid := range skillIDs {
		res, err := d.Exec("DELETE FROM group_skills WHERE group_id = ? AND skill_id = ?", groupID, sid)
		if err != nil {
			return removed, err
		}
		n, _ := res.RowsAffected()
		removed += int(n)
	}
	return removed, nil
}

// ---- Targets ----

func (d *DB) ListTargets() ([]TargetDir, error) {
	rows, err := d.Query(`
		SELECT td.id, td.path, td.label, td.created_at,
			(SELECT COUNT(*) FROM skills) as skill_count
		FROM target_dirs td ORDER BY td.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []TargetDir
	for rows.Next() {
		var t TargetDir
		if err := rows.Scan(&t.ID, &t.Path, &t.Label, &t.CreatedAt, &t.SkillCount); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func (d *DB) CreateTarget(path, label string) (*TargetDir, error) {
	res, err := d.Exec("INSERT INTO target_dirs (path, label) VALUES (?, ?)", path, label)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &TargetDir{ID: int(id), Path: path, Label: label}, nil
}

func (d *DB) DeleteTarget(id int) error {
	_, err := d.Exec("DELETE FROM target_dirs WHERE id = ?", id)
	return err
}

func (d *DB) GetTarget(id int) (*TargetDir, error) {
	var t TargetDir
	err := d.QueryRow("SELECT id, path, label, created_at FROM target_dirs WHERE id = ?", id).
		Scan(&t.ID, &t.Path, &t.Label, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}
