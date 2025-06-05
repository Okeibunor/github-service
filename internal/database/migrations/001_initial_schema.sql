-- Create repositories table
CREATE TABLE IF NOT EXISTS repositories (
    id BIGSERIAL PRIMARY KEY,
    github_id BIGINT UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    url VARCHAR(255) NOT NULL,
    language VARCHAR(100),
    forks_count INTEGER DEFAULT 0,
    stars_count INTEGER DEFAULT 0,
    open_issues_count INTEGER DEFAULT 0,
    watchers_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_commit_check TIMESTAMP WITH TIME ZONE,
    commits_since TIMESTAMP WITH TIME ZONE,
    created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create commits table
CREATE TABLE IF NOT EXISTS commits (
    id SERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    sha VARCHAR(40) NOT NULL,
    message TEXT NOT NULL,
    author_name VARCHAR(255) NOT NULL,
    author_email VARCHAR(255) NOT NULL,
    author_date TIMESTAMP WITH TIME ZONE NOT NULL,
    committer_name VARCHAR(255) NOT NULL,
    committer_email VARCHAR(255) NOT NULL,
    commit_date TIMESTAMP WITH TIME ZONE NOT NULL,
    url VARCHAR(255) NOT NULL,
    created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, sha)
);

-- Create indexes for better query performance
CREATE INDEX idx_commits_repository_date ON commits(repository_id, commit_date DESC);
CREATE INDEX idx_commits_author ON commits(author_name, author_email);
CREATE INDEX idx_repositories_name ON repositories(name, full_name); 