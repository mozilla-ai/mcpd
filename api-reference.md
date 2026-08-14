# API Reference

{% hint style="info" %}
Interactive reference generated from the mcpd OpenAPI specification.
{% endhint %}

## Health

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/health/servers" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/health/servers/{name}" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

## Prompts

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{name}/prompts" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{name}/prompts/{promptName}" method="post" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

## Resources

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{name}/resources" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{name}/resources/content" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{name}/resources/templates" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

## Servers

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

## Tools

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{name}/tools" method="get" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}

{% openapi-operation spec="mcpd-openapi-spec" path="/api/v1/servers/{server}/tools/{tool}" method="post" %}
[OpenAPI mcpd-openapi-spec](https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml)
{% endopenapi-operation %}
