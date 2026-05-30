package handler

import (
	"net/http"
	"strconv"

	"skill-web/db"

	"github.com/labstack/echo/v4"
)

type GroupHandler struct {
	DB *db.DB
}

func NewGroupHandler(database *db.DB) *GroupHandler {
	return &GroupHandler{DB: database}
}

// GET /api/groups
func (h *GroupHandler) List(c echo.Context) error {
	q := c.QueryParam("q")
	groups, err := h.DB.ListGroups(q)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if groups == nil {
		groups = []db.SkillGroup{}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"groups": groups,
		"total":  len(groups),
	})
}

// POST /api/groups
func (h *GroupHandler) Create(c echo.Context) error {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		SkillIDs    []string `json:"skill_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name required"})
	}

	group, err := h.DB.CreateGroup(req.Name, req.Description)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}

	if len(req.SkillIDs) > 0 {
		h.DB.AddGroupSkills(group.ID, req.SkillIDs)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"group": group,
	})
}

// GET /api/groups/:id
func (h *GroupHandler) Get(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	group, err := h.DB.GetGroup(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if group == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "group not found"})
	}

	skills, err := h.DB.GetGroupSkills(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if skills == nil {
		skills = []db.Skill{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"group": group,
		"skills": skills,
		"total": len(skills),
	})
}

// DELETE /api/groups/:id
func (h *GroupHandler) Delete(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := h.DB.DeleteGroup(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true})
}

// POST /api/groups/:id/skills
func (h *GroupHandler) AddSkills(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req struct {
		SkillIDs []string `json:"skill_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	added, err := h.DB.AddGroupSkills(id, req.SkillIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"group_id": id,
		"added":    added,
	})
}

// DELETE /api/groups/:id/skills
func (h *GroupHandler) RemoveSkills(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req struct {
		SkillIDs []string `json:"skill_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	removed, err := h.DB.RemoveGroupSkills(id, req.SkillIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"group_id": id,
		"removed":  removed,
	})
}
