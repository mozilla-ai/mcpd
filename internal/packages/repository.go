package packages

// Repository represents a source code repository with version verification.
// When used per-installation, care should be taken to verify that different repositories
// for the same server implement identical functionality to prevent security issues.
type Repository struct {
	// Type specifies the repository type (e.g., "git", "github").
	Type string `json:"type"`

	// URL is the repository URL where the source code is hosted.
	URL string `json:"url"`

	// Commit is the specific commit hash corresponding to the version tag.
	// This provides version verification and prevents tag manipulation attacks.
	Commit string `json:"commit,omitempty"`
}
