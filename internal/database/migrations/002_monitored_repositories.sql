-- Create monitored repositories table
CREATE TABLE IF NOT EXISTS monitored_repositories (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL UNIQUE,
    last_sync_time TIMESTAMP WITH TIME ZONE NOT NULL,
    sync_interval VARCHAR(50) NOT NULL, -- stored as duration string
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for active repositories
CREATE INDEX idx_monitored_repos_active ON monitored_repositories(is_active) WHERE is_active = true;

-- Down migration
-- DROP TABLE IF EXISTS monitored_repositories; 