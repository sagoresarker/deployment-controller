-- Database schema for Deployment Controller

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Deployments table with versioning support
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    app_name TEXT NOT NULL,
    docker_image TEXT NOT NULL,
    port INTEGER NOT NULL,
    env TEXT[] DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deployed_at TIMESTAMP WITH TIME ZONE,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'deploying', 'deployed', 'failed', 'rolled_back')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Composite unique constraint to ensure one active version per app per domain
    UNIQUE(domain, app_name, version)
);

-- Docker registry credentials table
CREATE TABLE docker_credentials (
    registry TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    password TEXT NOT NULL, -- Encrypted in production
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for better performance
CREATE INDEX idx_deployments_domain_app ON deployments(domain, app_name);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_updated_at ON deployments(updated_at DESC);
CREATE INDEX idx_deployments_request_id ON deployments(request_id);

-- View to get the latest version for each app
CREATE VIEW latest_deployments AS
SELECT DISTINCT ON (domain, app_name)
    id, request_id, domain, app_name, docker_image, port, env,
    version, updated_at, deployed_at, status, created_at
FROM deployments
ORDER BY domain, app_name, version DESC;

-- Function to get next version number for an app
CREATE OR REPLACE FUNCTION get_next_version(p_domain TEXT, p_app_name TEXT)
RETURNS INTEGER AS $$
DECLARE
    next_version INTEGER;
BEGIN
    SELECT COALESCE(MAX(version), 0) + 1
    INTO next_version
    FROM deployments
    WHERE domain = p_domain AND app_name = p_app_name;

    RETURN next_version;
END;
$$ LANGUAGE plpgsql;