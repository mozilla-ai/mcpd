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
	Export(path string) (map[string]string, error)
}

type UpsertResult string

const (
	Created UpsertResult = "created"
	Updated UpsertResult = "updated"
	Deleted UpsertResult = "deleted"
	Noop    UpsertResult = "noop"
)
