-- Create repositories table
CREATE TABLE IF NOT EXISTS repositories (
    id SERIAL PRIMARY KEY,
    github_id BIGINT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    description TEXT,
    url TEXT NOT NULL,
    language TEXT,
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
    repository_id BIGINT NOT NULL,
    sha TEXT NOT NULL,
    message TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_date TIMESTAMP WITH TIME ZONE NOT NULL,
    committer_name TEXT NOT NULL,
    committer_email TEXT NOT NULL,
    commit_date TIMESTAMP WITH TIME ZONE NOT NULL,
    url TEXT NOT NULL,
    created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE,
    UNIQUE(sha, repository_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_commits_repo_id ON commits (repository_id);
CREATE INDEX IF NOT EXISTS idx_commits_author ON commits (author_name);
CREATE INDEX IF NOT EXISTS idx_commits_date ON commits (commit_date);
CREATE INDEX IF NOT EXISTS idx_repositories_full_name ON repositories (full_name);
CREATE INDEX IF NOT EXISTS idx_repositories_github_id ON repositories (github_id);
CREATE INDEX IF NOT EXISTS idx_commits_sha ON commits (sha);
