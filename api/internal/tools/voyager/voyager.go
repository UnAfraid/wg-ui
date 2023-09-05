package voyager

import (
	"html/template"
	"net/http"
)

var page = template.Must(template.New("voyager").Parse(`<!DOCTYPE html>
<html>
  <head>
	<meta charset="utf-8" />
    <script src="https://cdn.jsdelivr.net/npm/react@16/umd/react.production.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/react-dom@16/umd/react-dom.production.min.js"></script>

    <link
		rel="stylesheet"
		href="https://cdn.jsdelivr.net/npm/graphql-voyager@{{ .version }}/dist/voyager.css"
		integrity="{{ .cssSRI }}" 
		crossorigin="anonymous"
    />
    <script src="https://cdn.jsdelivr.net/npm/graphql-voyager@{{ .version }}/dist/voyager.min.js"
		integrity="{{ .jsSRI }}" 
		crossorigin="anonymous"></script>
	<title>{{.title}}</title>
  </head>
  <body>
    <div id="voyager">Loading...</div>
    <script>
      function introspectionProvider(introspectionQuery) {
		  return fetch(location.protocol + '//' + location.host + '{{.endpoint}}', {
			method: 'post',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ query: introspectionQuery }),
		  }).then((response) => response.json());
      }

      GraphQLVoyager.init(document.getElementById('voyager'), {
        introspection: introspectionProvider,
      });
    </script>
  </body>
</html>
`))

func Handler(title string, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		err := page.Execute(w, map[string]string{
			"title":    title,
			"endpoint": endpoint,
			"version":  "1.0.1",
			"cssSRI":   "sha256-Ld1TmXe45lFBLvbZVG9REvEEBh3ckgGJZfhztcdCYtQ=",
			"jsSRI":    "sha256-cnsyiKhTVGWHy/Of4kZKJ4//abDXF1kzQsECnc+r/FM=",
		})
		if err != nil {
			panic(err)
		}
	}
}
