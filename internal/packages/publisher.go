package packages

// Publisher represents the organization or individual that published the server.
type Publisher struct {
	// Name is the name of the publisher (organization or individual).
	Name string `json:"name"`

	// URL is an optional link to the publisher's website or profile.
	URL string `json:"url,omitempty"`
}
