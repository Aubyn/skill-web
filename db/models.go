package db

type Skill struct {
	ID         string `json:"id"`
	SourcePath string `json:"source_path"`
	StorePath  string `json:"store_path"`
	SkillType  string `json:"skill_type"`
	CreatedAt  string `json:"created_at"`
}

type SkillGroup struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SkillCount  int    `json:"skill_count"`
	CreatedAt   string `json:"created_at"`
}

type TargetDir struct {
	ID         int    `json:"id"`
	Path       string `json:"path"`
	Label      string `json:"label"`
	SkillCount int    `json:"skill_count"`
	CreatedAt  string `json:"created_at"`
}
