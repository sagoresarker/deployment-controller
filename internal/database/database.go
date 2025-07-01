package database

import (
	"context"
	"fmt"
	"time"

	"deployment-controller/internal/config"
	"deployment-controller/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool
func New(cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set connection pool configuration
	poolConfig.MaxConns = int32(cfg.Database.MaxConns)
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
}

// CreateDeployment creates a new deployment record with versioning
func (db *DB) CreateDeployment(ctx context.Context, req models.DeploymentRequest, requestID string) (*models.Deployment, error) {
	// Start transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get next version number
	var version int
	err = tx.QueryRow(ctx, "SELECT get_next_version($1, $2)", req.Domain, req.AppName).Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version: %w", err)
	}

	// Set updated_at if not provided
	updatedAt := req.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	deployment := &models.Deployment{
		ID:          uuid.New(),
		RequestID:   requestID,
		Domain:      req.Domain,
		AppName:     req.AppName,
		DockerImage: req.DockerImage,
		Port:        req.Port,
		Env:         req.Env,
		Version:     version,
		UpdatedAt:   updatedAt,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	// Insert deployment
	query := `
		INSERT INTO deployments
		(id, request_id, domain, app_name, docker_image, port, env, version, updated_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = tx.Exec(ctx, query,
		deployment.ID, deployment.RequestID, deployment.Domain, deployment.AppName,
		deployment.DockerImage, deployment.Port, deployment.Env, deployment.Version,
		deployment.UpdatedAt, deployment.Status, deployment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert deployment: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return deployment, nil
}

// GetDeployment gets a deployment by ID
func (db *DB) GetDeployment(ctx context.Context, id uuid.UUID) (*models.Deployment, error) {
	deployment := &models.Deployment{}
	query := `
		SELECT id, request_id, domain, app_name, docker_image, port, env, version,
		       updated_at, deployed_at, status, created_at
		FROM deployments
		WHERE id = $1
	`
	row := db.Pool.QueryRow(ctx, query, id)
	err := row.Scan(
		&deployment.ID, &deployment.RequestID, &deployment.Domain, &deployment.AppName,
		&deployment.DockerImage, &deployment.Port, &deployment.Env, &deployment.Version,
		&deployment.UpdatedAt, &deployment.DeployedAt, &deployment.Status, &deployment.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("deployment not found")
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return deployment, nil
}

// GetLatestDeployments gets the latest version of all deployments
func (db *DB) GetLatestDeployments(ctx context.Context) ([]models.Deployment, error) {
	query := `
		SELECT id, request_id, domain, app_name, docker_image, port, env, version,
		       updated_at, deployed_at, status, created_at
		FROM latest_deployments
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployments: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		var deployment models.Deployment
		err := rows.Scan(
			&deployment.ID, &deployment.RequestID, &deployment.Domain, &deployment.AppName,
			&deployment.DockerImage, &deployment.Port, &deployment.Env, &deployment.Version,
			&deployment.UpdatedAt, &deployment.DeployedAt, &deployment.Status, &deployment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// UpdateDeploymentStatus updates the status of a deployment
func (db *DB) UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status string, deployedAt *time.Time) error {
	query := `
		UPDATE deployments
		SET status = $1, deployed_at = $2
		WHERE id = $3
	`
	_, err := db.Pool.Exec(ctx, query, status, deployedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	return nil
}

// StoreRegistryCredential stores Docker registry credentials
func (db *DB) StoreRegistryCredential(ctx context.Context, cred models.RegistryCredentialRequest) error {
	query := `
		INSERT INTO docker_credentials (registry, username, password, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (registry)
		DO UPDATE SET username = $2, password = $3, updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, cred.Registry, cred.Username, cred.Password)
	if err != nil {
		return fmt.Errorf("failed to store registry credential: %w", err)
	}

	return nil
}

// GetRegistryCredential gets Docker registry credentials
func (db *DB) GetRegistryCredential(ctx context.Context, registry string) (*models.RegistryCredentialResponse, error) {
	cred := &models.RegistryCredentialResponse{}
	query := `
		SELECT registry, username, password
		FROM docker_credentials
		WHERE registry = $1
	`
	row := db.Pool.QueryRow(ctx, query, registry)
	err := row.Scan(&cred.Registry, &cred.Username, &cred.Password)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("registry credential not found")
		}
		return nil, fmt.Errorf("failed to get registry credential: %w", err)
	}

	return cred, nil
}

// GetDeploymentStats gets deployment statistics
func (db *DB) GetDeploymentStats(ctx context.Context) (*models.DeploymentStats, error) {
	stats := &models.DeploymentStats{}
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'deployed' THEN 1 END) as deployed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM latest_deployments
	`
	row := db.Pool.QueryRow(ctx, query)
	err := row.Scan(&stats.TotalDeployments, &stats.PendingCount, &stats.DeployedCount, &stats.FailedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment stats: %w", err)
	}

	return stats, nil
}
