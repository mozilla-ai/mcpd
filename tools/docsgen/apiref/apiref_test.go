package apiref

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderGroupsByTagAlphabetically(t *testing.T) {
	t.Parallel()

	ops := []Operation{
		{Tag: "Servers", Method: "get", Path: "/api/v1/servers"},
		{Tag: "Health", Method: "get", Path: "/api/v1/health/servers"},
	}

	want := "# API Reference\n\n" +
		"{% hint style=\"info\" %}\n" +
		"Interactive reference generated from the mcpd OpenAPI specification.\n" +
		"{% endhint %}\n\n" +
		"## Health\n\n" +
		"{% openapi-operation spec=\"mcpd-openapi-spec\" path=\"/api/v1/health/servers\" method=\"get\" %}\n" +
		"[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)\n" +
		"{% endopenapi-operation %}\n\n" +
		"## Servers\n\n" +
		"{% openapi-operation spec=\"mcpd-openapi-spec\" path=\"/api/v1/servers\" method=\"get\" %}\n" +
		"[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)\n" +
		"{% endopenapi-operation %}\n"

	require.Equal(t, want, Render(ops))
}

func TestRenderSortsWithinTagByPathThenMethod(t *testing.T) {
	t.Parallel()

	ops := []Operation{
		{Tag: "Tools", Method: "post", Path: "/api/v1/servers/{server}/tools/{tool}"},
		{Tag: "Tools", Method: "get", Path: "/api/v1/servers/{name}/tools"},
	}

	got := Render(ops)

	require.Less(t,
		strings.Index(got, `path="/api/v1/servers/{name}/tools"`),
		strings.Index(got, `path="/api/v1/servers/{server}/tools/{tool}"`),
	)
}

func TestRenderSortsByMethodWhenPathsEqual(t *testing.T) {
	t.Parallel()

	ops := []Operation{
		{Tag: "Tools", Method: "post", Path: "/api/v1/servers/{name}/tools"},
		{Tag: "Tools", Method: "get", Path: "/api/v1/servers/{name}/tools"},
	}

	got := Render(ops)

	require.Less(t, strings.Index(got, `method="get"`), strings.Index(got, `method="post"`))
}

func TestRenderGroupsUntaggedOperationsUnderOther(t *testing.T) {
	t.Parallel()

	got := Render([]Operation{{Tag: "", Method: "get", Path: "/api/v1/servers"}})

	require.Contains(t, got, "## Other\n")
}
