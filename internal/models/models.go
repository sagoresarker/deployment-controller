package models

import (
	"time"

	"github.com/google/uuid"
)

// DeploymentRequest represents the incoming deployment request
type DeploymentRequest struct {
	Domain      string    `json:"domain" binding:"required"`
	AppName     string    `json:"app_name" binding:"required"`
	DockerImage string    `json:"docker_image" binding:"required"`
	Port        int       `json:"port" binding:"required,min=1,max=65535"`
	Env         []string  `json:"env"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DeploymentPushRequest represents the array of deployment changes
type DeploymentPushRequest []DeploymentRequest

// Deployment represents a deployment record in the database
type Deployment struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	RequestID   string     `json:"request_id" db:"request_id"`
	Domain      string     `json:"domain" db:"domain"`
	AppName     string     `json:"app_name" db:"app_name"`
	DockerImage string     `json:"docker_image" db:"docker_image"`
	Port        int        `json:"port" db:"port"`
	Env         []string   `json:"env" db:"env"`
	Version     int        `json:"version" db:"version"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeployedAt  *time.Time `json:"deployed_at,omitempty" db:"deployed_at"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// RegistryCredential represents Docker registry credentials
type RegistryCredential struct {
	Registry  string    `json:"registry" db:"registry"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"password" db:"password"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RegistryCredentialRequest represents the request to store registry credentials
type RegistryCredentialRequest struct {
	Registry string `json:"registry" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegistryCredentialResponse represents the response when getting registry credentials
type RegistryCredentialResponse struct {
	Registry string `json:"registry"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// DeploymentStats represents deployment statistics
type DeploymentStats struct {
	TotalDeployments int `json:"total_deployments"`
	PendingCount     int `json:"pending_count"`
	DeployedCount    int `json:"deployed_count"`
	FailedCount      int `json:"failed_count"`
}
