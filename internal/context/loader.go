package context

var (
	_ Loader   = (*DefaultLoader)(nil)
	_ Modifier = (*ExecutionContextConfig)(nil)
)

type Loader interface {
	Load(path string) (Modifier, error)
}

type Modifier interface {
	AddServer(ec ServerExecutionContext) error
	RemoveServer(name string) error
	ListServers() map[string]ServerExecutionContext
}
