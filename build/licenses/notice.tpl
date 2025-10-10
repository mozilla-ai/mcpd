# NOTICE

This product includes third-party software. The following dependencies are included in this distribution, along with their licenses:

{{ range . }}
## {{ .Name }} ({{ .Version }})

License: [{{ .LicenseName }}]({{ .LicenseURL }})

{{ .LicenseText }}

{{ end }}