-- Schema for GitHub Service Database

-- Repositories table to store repository metadata
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

-- Commits table to store commit information
CREATE TABLE IF NOT EXISTS commits (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL,
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
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(repository_id, sha)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_commits_repo_date ON commits(repository_id, commit_date DESC);
CREATE INDEX IF NOT EXISTS idx_commits_author ON commits(author_name, author_email);
CREATE INDEX IF NOT EXISTS idx_repositories_name ON repositories(name, full_name); 