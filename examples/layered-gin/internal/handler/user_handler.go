package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/dnahilman/goten/examples/layered-gin/internal/service"
)

type UserHandler struct{ svc *service.UserService }

func NewUserHandler(s *service.UserService) *UserHandler { return &UserHandler{svc: s} }

// Register binds the user-profile routes onto an authenticated route group.
func (h *UserHandler) Register(r *gin.RouterGroup) {
	r.GET("/me", h.getMe)
	r.POST("/me", h.createProfile)
	r.PATCH("/me/phone", h.updatePhone)
}

func (h *UserHandler) getMe(c *gin.Context) {
	u, err := h.svc.GetProfile(c.Request.Context(), AuthUserID(c))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not created yet — POST /api/me first"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) createProfile(c *gin.Context) {
	var body struct {
		FullName string `json:"full_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.svc.CreateProfile(c.Request.Context(), AuthUserID(c), body.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func (h *UserHandler) updatePhone(c *gin.Context) {
	var body struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdatePhone(c.Request.Context(), AuthUserID(c), body.Phone); err != nil {
		if errors.Is(err, service.ErrInvalidPhone) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
