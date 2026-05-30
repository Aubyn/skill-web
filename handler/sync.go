package handler

import (
	"net/http"

	"skill-web/db"
	"skill-web/syncer"

	"github.com/labstack/echo/v4"
)

type SyncHandler struct {
	DB       *db.DB
	StoreDir string
}

func NewSyncHandler(database *db.DB, storeDir string) *SyncHandler {
	return &SyncHandler{DB: database, StoreDir: storeDir}
}

// POST /api/sync
func (h *SyncHandler) Sync(c echo.Context) error {
	var req struct {
		SkillIDs  []string `json:"skill_ids"`
		GroupIDs  []int    `json:"group_ids"`
		TargetIDs []int    `json:"target_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if len(req.TargetIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "target_ids required"})
	}

	// Resolve group IDs to skill IDs
	skillIDs := make([]string, 0, len(req.SkillIDs))
	skillIDs = append(skillIDs, req.SkillIDs...)

	seen := map[string]bool{}
	for _, sid := range req.SkillIDs {
		seen[sid] = true
	}

	for _, gid := range req.GroupIDs {
		skills, err := h.DB.GetGroupSkills(gid)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		for _, s := range skills {
			if !seen[s.ID] {
				skillIDs = append(skillIDs, s.ID)
				seen[s.ID] = true
			}
		}
	}

	if len(skillIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no skills to sync"})
	}

	// Resolve target paths
	var targetPaths []string
	for _, tid := range req.TargetIDs {
		target, err := h.DB.GetTarget(tid)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if target == nil {
			continue
		}

		targetPath := resolvePath(target.Path)
		targetPaths = append(targetPaths, targetPath)
	}

	if len(targetPaths) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no valid targets"})
	}

	result, err := syncer.Sync(h.StoreDir, skillIDs, targetPaths)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}
