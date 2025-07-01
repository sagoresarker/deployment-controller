package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"deployment-controller/internal/database"
	"deployment-controller/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	db     *database.DB
	logger *slog.Logger
}

// New creates a new handler instance
func New(db *database.DB, logger *slog.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

// Push handles POST /api/v1/push - receives deployment changes
func (h *Handler) Push(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	var deploymentRequests models.DeploymentPushRequest
	if err := c.ShouldBindJSON(&deploymentRequests); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(deploymentRequests) == 0 {
		h.logger.Error("Empty deployment request")
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "At least one deployment is required",
		})
		return
	}

	// Generate a unique request ID for this batch
	requestID := uuid.New().String()
	h.logger.Info("Processing deployment push",
		"request_id", requestID,
		"count", len(deploymentRequests))

	var createdDeployments []models.Deployment
	var failedDeployments []map[string]interface{}

	// Process each deployment request
	for i, req := range deploymentRequests {
		deployment, err := h.db.CreateDeployment(ctx, req, requestID)
		if err != nil {
			h.logger.Error("Failed to create deployment",
				"error", err,
				"domain", req.Domain,
				"app_name", req.AppName)

			failedDeployments = append(failedDeployments, map[string]interface{}{
				"index":    i,
				"domain":   req.Domain,
				"app_name": req.AppName,
				"error":    err.Error(),
			})
			continue
		}

		createdDeployments = append(createdDeployments, *deployment)
		h.logger.Info("Created deployment",
			"deployment_id", deployment.ID,
			"domain", deployment.Domain,
			"app_name", deployment.AppName,
			"version", deployment.Version)
	}

	// Prepare response
	responseData := map[string]interface{}{
		"request_id":          requestID,
		"processed_count":     len(createdDeployments),
		"failed_count":        len(failedDeployments),
		"created_deployments": createdDeployments,
	}

	if len(failedDeployments) > 0 {
		responseData["failed_deployments"] = failedDeployments
	}

	statusCode := http.StatusCreated
	if len(failedDeployments) > 0 && len(createdDeployments) == 0 {
		statusCode = http.StatusBadRequest
	} else if len(failedDeployments) > 0 {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, models.APIResponse{
		Success: len(createdDeployments) > 0,
		Message: "Deployment push processed",
		Data:    responseData,
	})
}

// StoreRegistryCredential handles POST /api/v1/registry
func (h *Handler) StoreRegistryCredential(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req models.RegistryCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid registry credential request", "error", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.db.StoreRegistryCredential(ctx, req); err != nil {
		h.logger.Error("Failed to store registry credential",
			"error", err,
			"registry", req.Registry)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to store registry credential",
		})
		return
	}

	h.logger.Info("Stored registry credential", "registry", req.Registry)
	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Registry credential stored successfully",
	})
}

// GetRegistryCredential handles GET /api/v1/registry
func (h *Handler) GetRegistryCredential(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	registry := c.Query("registry")
	if registry == "" {
		h.logger.Error("Missing registry parameter")
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "registry parameter is required",
		})
		return
	}

	cred, err := h.db.GetRegistryCredential(ctx, registry)
	if err != nil {
		h.logger.Error("Failed to get registry credential",
			"error", err,
			"registry", registry)

		if err.Error() == "registry credential not found" {
			c.JSON(http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "Registry credential not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to get registry credential",
		})
		return
	}

	h.logger.Info("Retrieved registry credential", "registry", registry)
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    cred,
	})
}

// GetDeployments handles GET /api/v1/deployments
func (h *Handler) GetDeployments(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	deployments, err := h.db.GetLatestDeployments(ctx)
	if err != nil {
		h.logger.Error("Failed to get deployments", "error", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to get deployments",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    deployments,
	})
}

// GetDeployment handles GET /api/v1/deployments/:id
func (h *Handler) GetDeployment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid deployment ID", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid deployment ID",
		})
		return
	}

	deployment, err := h.db.GetDeployment(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get deployment", "error", err, "id", id)

		if err.Error() == "deployment not found" {
			c.JSON(http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "Deployment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to get deployment",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    deployment,
	})
}

// UpdateDeploymentStatus handles PATCH /api/v1/deployments/:id/status
func (h *Handler) UpdateDeploymentStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid deployment ID", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid deployment ID",
		})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid status update request", "error", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending":     true,
		"deploying":   true,
		"deployed":    true,
		"failed":      true,
		"rolled_back": true,
	}

	if !validStatuses[req.Status] {
		h.logger.Error("Invalid status", "status", req.Status)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid status. Must be one of: pending, deploying, deployed, failed, rolled_back",
		})
		return
	}

	var deployedAt *time.Time
	if req.Status == "deployed" {
		now := time.Now()
		deployedAt = &now
	}

	if err := h.db.UpdateDeploymentStatus(ctx, id, req.Status, deployedAt); err != nil {
		h.logger.Error("Failed to update deployment status",
			"error", err,
			"id", id,
			"status", req.Status)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to update deployment status",
		})
		return
	}

	h.logger.Info("Updated deployment status",
		"id", id,
		"status", req.Status)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Deployment status updated successfully",
	})
}

// GetStats handles GET /api/v1/stats
func (h *Handler) GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	stats, err := h.db.GetDeploymentStats(ctx)
	if err != nil {
		h.logger.Error("Failed to get deployment stats", "error", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to get deployment stats",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

// HealthCheck handles GET /healthz
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Test database connection
	if err := h.db.Pool.Ping(ctx); err != nil {
		h.logger.Error("Database health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, models.APIResponse{
			Success: false,
			Error:   "Database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Service is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1.0.0",
		},
	})
}
