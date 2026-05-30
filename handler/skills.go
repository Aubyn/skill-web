package handler

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"skill-web/db"
	"skill-web/scanner"

	"github.com/labstack/echo/v4"
)

type SkillHandler struct {
	DB       *db.DB
	StoreDir string
}

func NewSkillHandler(database *db.DB, storeDir string) *SkillHandler {
	return &SkillHandler{DB: database, StoreDir: storeDir}
}

// GET /api/skills
func (h *SkillHandler) List(c echo.Context) error {
	q := c.QueryParam("q")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 {
		pageSize = 50
	} else if pageSize > 100 {
		pageSize = 100
	}

	skills, total, err := h.DB.ListSkills(q, page, pageSize)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if skills == nil {
		skills = []db.Skill{}
	}

	totalPages := (total + pageSize - 1) / pageSize

	return c.JSON(http.StatusOK, map[string]any{
		"skills":      skills,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// GET /api/skills/:id
func (h *SkillHandler) Get(c echo.Context) error {
	id := c.Param("id")
	skill, err := h.DB.GetSkill(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if skill == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "skill not found"})
	}

	// List files in the skill directory
	var files []string
	filepath.Walk(skill.StorePath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(skill.StorePath, path)
		if rel != "." {
			files = append(files, rel)
		}
		return nil
	})
	if files == nil {
		files = []string{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"skill": skill,
		"files": files,
	})
}

// DELETE /api/skills/:id
func (h *SkillHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	skill, err := h.DB.GetSkill(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if skill == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "skill not found"})
	}

	// Remove from disk
	os.RemoveAll(skill.StorePath)

	// Remove from DB
	if err := h.DB.DeleteSkill(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success":      true,
		"removed_path": skill.StorePath,
	})
}

// GET /api/scan-preview
func (h *SkillHandler) ScanPreview(c echo.Context) error {
	dir := c.QueryParam("dir")
	if dir == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "dir parameter required"})
	}

	// Handle zip: extract to temp dir
	scanDir, cleanup, err := resolveScanDir(dir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Get existing skill IDs
	existing := map[string]bool{}
	skills, _, _ := h.DB.ListSkills("", 1, 10000)
	for _, s := range skills {
		existing[s.ID] = true
	}

	entries, conflicts, err := scanner.Scan(scanDir, existing, 20)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"found":                 entries,
		"conflicts_with_existing": conflicts,
		"total":                 len(entries),
	})
}

// extractDescription reads the YAML frontmatter from a skill's markdown file
// and returns the description field value.
func extractDescription(storeDir, id string) string {
	// Try SKILL.md or <id>.md
	candidates := []string{
		filepath.Join(storeDir, id, "SKILL.md"),
		filepath.Join(storeDir, id, id+".md"),
	}
	for _, path := range candidates {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		// Expect first line to be "---"
		if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
			continue
		}
		// Scan until closing "---"
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "---" {
				break
			}
			if strings.HasPrefix(line, "description:") || strings.HasPrefix(line, "description :") {
				val := strings.TrimSpace(line[strings.Index(line, ":")+1:])
				val = strings.Trim(val, ` "'`)

				// Handle YAML block scalar: description: >  or description: |
				if val == "" || val == ">" || val == "|" || val == ">-" || val == "|-" || val == ">+" || val == "|+" {
					var parts []string
					for scanner.Scan() {
						cont := scanner.Text()
						if cont == "" || (cont[0] != ' ' && cont[0] != '\t') {
							break
						}
						parts = append(parts, strings.TrimSpace(cont))
					}
					val = strings.Join(parts, " ")
				}

				if val != "" {
					return val
				}
			}
		}
	}
	return ""
}

// POST /api/import
func (h *SkillHandler) Import(c echo.Context) error {
	var req struct {
		Dir      string   `json:"dir"`
		SkillIDs []string `json:"skill_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Dir == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "dir required"})
	}

	// Handle zip: extract to temp dir
	scanDir, cleanup, err := resolveScanDir(req.Dir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Get existing skill IDs
	existing := map[string]bool{}
	skills, _, _ := h.DB.ListSkills("", 1, 10000)
	for _, s := range skills {
		existing[s.ID] = true
	}

	entries, _, err := scanner.Scan(scanDir, existing, 20)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Filter by skill_ids if provided
	if len(req.SkillIDs) > 0 {
		want := map[string]bool{}
		for _, id := range req.SkillIDs {
			want[id] = true
		}
		var filtered []scanner.Entry
		for _, e := range entries {
			if want[e.ID] {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	var imported []map[string]string
	var conflicts []map[string]string

	for _, e := range entries {
		destDir := filepath.Join(h.StoreDir, e.ID)

		// Copy to store
		if err := copyDir(e.SourcePath, destDir); err != nil {
			continue
		}

		desc := extractDescription(h.StoreDir, e.ID)
		skill := &db.Skill{
			ID:          e.ID,
			SourcePath:  e.SourcePath,
			StorePath:   destDir,
			SkillType:   "dir",
			Description: desc,
		}
		if err := h.DB.UpsertSkill(skill); err != nil {
			continue
		}

		imported = append(imported, map[string]string{
			"id":         e.ID,
			"source":     e.SourcePath,
			"store_path": destDir,
		})

		if existing[e.ID] {
			conflicts = append(conflicts, map[string]string{
				"name":   e.ID,
				"action": "overwritten",
			})
		}
	}

	if imported == nil {
		imported = []map[string]string{}
	}
	if conflicts == nil {
		conflicts = []map[string]string{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"imported":  imported,
		"conflicts": conflicts,
		"total":     len(imported),
	})
}

// resolveScanDir returns the directory to scan. If path is a .zip file,
// it extracts to a temp dir and returns a cleanup function.
func resolveScanDir(path string) (scanDir string, cleanup func(), err error) {
	if !strings.HasSuffix(strings.ToLower(path), ".zip") {
		return path, nil, nil
	}

	tmpDir, err := os.MkdirTemp(os.TempDir(), "skill-web-zip-*")
	if err != nil {
		return "", nil, fmt.Errorf("cannot create temp dir: %w", err)
	}

	if err := extractZip(path, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("cannot extract zip: %w", err)
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }, nil
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Zip slip protection
		target := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	// Remove destination if exists
	os.RemoveAll(dst)

	// Use filepath.Walk to copy all contents
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, fi.Mode())
		}

		destPath := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(destPath, fi.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Create parent dirs
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(destPath, data, fi.Mode())
	})
}

// Helper to safely resolve symlinks for display
func resolveSymlink(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return path
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	return target
}

// Ensure StoreDir exists
func (h *SkillHandler) EnsureStoreDir() error {
	return os.MkdirAll(h.StoreDir, 0755)
}

// Utility: check if a path looks like one we manage
func isManagedSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

// Prevent unused import warnings
var _ = resolveSymlink
var _ = isManagedSymlink
