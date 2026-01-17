package api

import (
	"fmt"
	"net/http"
)

// ScalarHandler returns an HTTP handler that serves the Scalar API documentation UI.
func ScalarHandler(specURL, title, description string) http.Handler {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>%s - API Documentation</title>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<style>
		body {
			margin: 0;
			padding: 0;
		}
	</style>
</head>
<body>
	<script id="api-reference" data-url="%s"></script>
	<script>
		var configuration = {
			theme: 'purple',
			layout: 'modern',
			showSidebar: true,
			hideModels: false,
			hideDownloadButton: false,
			hideTestRequestButton: false,
			darkMode: true,
			forceDarkModeState: 'dark',
			metaData: {
				title: '%s',
				description: '%s'
			},
			servers: [
				{
					url: window.location.origin,
					description: 'Current server'
				}
			]
		}
		document.getElementById('api-reference').dataset.configuration = JSON.stringify(configuration)
	</script>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`, title, specURL, title, description)

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})
}
