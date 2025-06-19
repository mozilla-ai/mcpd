package packages

type Installation struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Package     string            `json:"package,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Description string            `json:"description,omitempty"`
	Recommended bool              `json:"recommended,omitempty"`
}
