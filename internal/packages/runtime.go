package packages

// RuntimeUsage describes how an argument is used in a specific runtime
type RuntimeUsage struct {
	ActualName string `json:"actualName"` // The actual name used (e.g., "--local-timezone" for TZ)
	Context    string `json:"context"`    // Additional context about usage
}
