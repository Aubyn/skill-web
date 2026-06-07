package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"skill-web/db"

	"github.com/labstack/echo/v4"
)

type TargetHandler struct {
	DB *db.DB
}

// resolvePath expands ~ to home dir and makes relative paths absolute
func resolvePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	} else if !filepath.IsAbs(path) {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path)
	}
	return path
}

// GET /api/targets/:id/skills — list deployed skills in a target directory
func (h *TargetHandler) ListSkills(c echo.Context) error {
	id, err := parseIntParam(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	target, err := h.DB.GetTarget(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if target == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "target not found"})
	}

	targetPath := resolvePath(target.Path)

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]any{
			"skills": []string{},
			"total":  0,
			"error":  "目录不存在或不可读: " + err.Error(),
		})
	}

	var skills []string
	for _, e := range entries {
		fullPath := filepath.Join(targetPath, e.Name())
		fi, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			skills = append(skills, e.Name())
		}
	}
	if skills == nil {
		skills = []string{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"skills": skills,
		"total":  len(skills),
	})
}

// POST /api/targets/:id/clear — remove all deployed skills from a target directory
func (h *TargetHandler) Clear(c echo.Context) error {
	id, err := parseIntParam(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	target, err := h.DB.GetTarget(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if target == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "target not found"})
	}

	targetPath := resolvePath(target.Path)

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "目录不存在或不可读: " + err.Error()})
	}

	// Backup existing entries
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("2006-01-02T150405")
	backupDir := filepath.Join(home, ".skill-web", "backups", "clear-"+ts, filepath.Base(targetPath))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "创建备份目录失败: " + err.Error()})
	}

	var removed []string
	var failed []string

	for _, e := range entries {
		fullPath := filepath.Join(targetPath, e.Name())
		fi, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 || fi.IsDir() {
			backupPath := filepath.Join(backupDir, e.Name())

			// Try rename first (fails if cross-filesystem)
			err := os.Rename(fullPath, backupPath)
			if err != nil {
				// Cross-filesystem fallback: copy + delete
				var copyErr error
				if fi.IsDir() {
					copyErr = copyDir(fullPath, backupPath)
				} else {
					copyErr = copySymlink(fullPath, backupPath)
				}
				if copyErr != nil {
					failed = append(failed, e.Name())
					continue
				}
				os.RemoveAll(fullPath)
			}
			removed = append(removed, e.Name())
		}
	}

	if failed == nil {
		failed = []string{}
	}
	if removed == nil {
		removed = []string{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"removed":     removed,
		"failed":      failed,
		"total":       len(removed),
		"backup_path": backupDir,
	})
}

// GET /api/targets/exists-check — check if target directories exist on disk
func (h *TargetHandler) ExistsCheck(c echo.Context) error {
	idsParam := c.QueryParam("ids")
	if idsParam == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ids param required"})
	}

	parts := strings.Split(idsParam, ",")
	type Result struct {
		ID     int    `json:"id"`
		Exists bool   `json:"exists"`
		Path   string `json:"path"`
		Error  string `json:"error,omitempty"`
	}
	results := []Result{}

	for _, p := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			continue
		}
		target, err := h.DB.GetTarget(id)
		if err != nil {
			results = append(results, Result{ID: id, Exists: false, Error: "db lookup failed"})
			continue
		}
		if target == nil {
			results = append(results, Result{ID: id, Exists: false, Error: "not found"})
			continue
		}
		_, err = os.Stat(resolvePath(target.Path))
		results = append(results, Result{
			ID:     id,
			Exists: err == nil,
			Path:   target.Path,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{"results": results})
}

// copySymlink copies a symlink by re-creating it at the destination,
// preserving the symlink target path.
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(target, dst)
}

func NewTargetHandler(database *db.DB) *TargetHandler {
	return &TargetHandler{DB: database}
}

// GET /api/targets
func (h *TargetHandler) List(c echo.Context) error {
	targets, err := h.DB.ListTargets()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if targets == nil {
		targets = []db.TargetDir{}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"targets": targets,
		"total":   len(targets),
	})
}

// POST /api/targets
func (h *TargetHandler) Create(c echo.Context) error {
	var req struct {
		Path  string `json:"path"`
		Label string `json:"label"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Path == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "path required"})
	}

	target, err := h.DB.CreateTarget(req.Path, req.Label)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"target": target,
	})
}

// DELETE /api/targets/:id
func (h *TargetHandler) Delete(c echo.Context) error {
	id, err := parseIntParam(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := h.DB.DeleteTarget(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true})
}

func parseIntParam(s string) (int, error) {
	var id int
	_, err := scanInt(s, &id)
	return id, err
}

func scanInt(s string, v *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid integer")
		}
		n = n*10 + int(c-'0')
	}
	*v = n
	return n, nil
}
