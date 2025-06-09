package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrDuplicate is returned when attempting to create a duplicate resource
	ErrDuplicate = errors.New("resource already exists")

	// ErrInvalidInput is returned when the input parameters are invalid
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrRateLimit is returned when GitHub API rate limit is exceeded
	ErrRateLimit = errors.New("github api rate limit exceeded")

	// ErrGitHubAPI is returned when GitHub API returns an error
	ErrGitHubAPI = errors.New("github api error")

	// ErrDatabase is returned when a database operation fails
	ErrDatabase = errors.New("database error")

	// ErrUnauthorized is returned when authentication fails
	ErrUnauthorized = errors.New("unauthorized")
)

// RepositoryError represents an error related to repository operations
type RepositoryError struct {
	Owner string
	Name  string
	Op    string
	Err   error
}

func (e *RepositoryError) Error() string {
	if e.Owner == "" || e.Name == "" {
		return fmt.Sprintf("repository operation %s failed: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("repository operation %s failed for %s/%s: %v", e.Op, e.Owner, e.Name, e.Err)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// NewRepositoryError creates a new RepositoryError
func NewRepositoryError(owner, name, op string, err error) error {
	return &RepositoryError{
		Owner: owner,
		Name:  name,
		Op:    op,
		Err:   err,
	}
}

// CommitError represents an error related to commit operations
type CommitError struct {
	RepositoryID int64
	SHA          string
	Op           string
	Err          error
}

func (e *CommitError) Error() string {
	if e.SHA == "" {
		return fmt.Sprintf("commit operation %s failed for repository %d: %v", e.Op, e.RepositoryID, e.Err)
	}
	return fmt.Sprintf("commit operation %s failed for repository %d, commit %s: %v", e.Op, e.RepositoryID, e.SHA, e.Err)
}

func (e *CommitError) Unwrap() error {
	return e.Err
}

// NewCommitError creates a new CommitError
func NewCommitError(repoID int64, sha, op string, err error) error {
	return &CommitError{
		RepositoryID: repoID,
		SHA:          sha,
		Op:           op,
		Err:          err,
	}
}

// DatabaseError represents a database operation error
type DatabaseError struct {
	Op  string
	Err error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database operation %s failed: %v", e.Op, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// NewDatabaseError creates a new DatabaseError
func NewDatabaseError(op string, err error) error {
	return &DatabaseError{
		Op:  op,
		Err: err,
	}
}

// GitHubError represents a GitHub API error
type GitHubError struct {
	Op      string
	Request string
	Err     error
}

func (e *GitHubError) Error() string {
	return fmt.Sprintf("github api operation %s failed for request %s: %v", e.Op, e.Request, e.Err)
}

func (e *GitHubError) Unwrap() error {
	return e.Err
}

// NewGitHubError creates a new GitHubError
func NewGitHubError(op, request string, err error) error {
	return &GitHubError{
		Op:      op,
		Request: request,
		Err:     err,
	}
}

// Is checks if the target error matches any of our custom errors
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
