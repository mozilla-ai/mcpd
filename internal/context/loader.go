package context

var (
	_ Loader   = (*DefaultLoader)(nil)
	_ Modifier = (*ExecutionContextConfig)(nil)
)

type Loader interface {
	Load(path string) (Modifier, error)
}

type Modifier interface {
	Get(name string) (ServerExecutionContext, bool)
	Upsert(ctx ServerExecutionContext) (UpsertResult, error)
	List() []ServerExecutionContext
}

type Exporter interface {
	// Export handles exporting runtime execution context data to the specified path.
	// It returns a map which can be used by the caller, and is intended to contain
	// additional information such as the contract data for processing.
	Export(path string) (map[string]string, error)
}

type UpsertResult string

const (
	Created UpsertResult = "created"
	Updated UpsertResult = "updated"
	Deleted UpsertResult = "deleted"
	Noop    UpsertResult = "noop"
)
