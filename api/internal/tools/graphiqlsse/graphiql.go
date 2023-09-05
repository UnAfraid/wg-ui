package graphiqlsse

import (
	"html/template"
	"net/http"
	"net/url"
)

var page = template.Must(template.New("graphiql").Parse(`<!DOCTYPE html>
<html>
  <head>
  	<meta charset="utf-8">
  	<title>{{.title}}</title>
	<style>
		body {
			height: 100%;
			margin: 0;
			width: 100%;
			overflow: hidden;
		}

		#graphiql {
			height: 100vh;
		}
	</style>
	<script
		src="https://cdn.jsdelivr.net/npm/react@18.2.0/umd/react.production.min.js"
		integrity="{{.reactSRI}}"
		crossorigin="anonymous"
	></script>
	<script
		src="https://cdn.jsdelivr.net/npm/react-dom@18.2.0/umd/react-dom.production.min.js"
		integrity="{{.reactDOMSRI}}"
		crossorigin="anonymous"
	></script>
    <link
		rel="stylesheet"
		href="https://cdn.jsdelivr.net/npm/graphiql@{{.version}}/graphiql.min.css"
		integrity="{{.cssSRI}}"
		crossorigin="anonymous"
	/>
  </head>
  <body>
    <div id="graphiql">Loading...</div>
	<script
		src="https://cdn.jsdelivr.net/npm/graphiql@{{.version}}/graphiql.min.js"
		integrity="{{.jsSRI}}"
		crossorigin="anonymous"
	></script>
    <script
      src="https://unpkg.com/graphql-sse/umd/graphql-sse.js"
      type="application/javascript"
    ></script>
    <script>
{{- if .endpointIsAbsolute}}
      const url = {{.endpoint}};
{{- else}}
      const url = location.protocol + '//' + location.host + {{.endpoint}};
{{- end}}
      const sseClient = graphqlSse.createClient({
      	singleConnection: false,
        url: url,
        retryAttempts: 0,
		headers: () => {
			const headers = localStorage.getItem("graphiql:headers");
			return headers ? JSON.parse(headers) : {};
		},
      });
      function fetcher(payload) {
        let deferred = null;
        const pending = [];
        let throwMe = null,
        done = false;
        const dispose = sseClient.subscribe(payload, {
          next: (data) => {
            pending.push(data);
            deferred?.resolve(false);
          },
          error: (err) => {
            throwMe = err;
            deferred?.reject(throwMe);
          },
          complete: () => {
            done = true;
            deferred?.resolve(true);
          },
        });
        return {
          [Symbol.asyncIterator]() {
            return this;
          },
          async next() {
            if (done) {
              return { done: true, value: undefined };
            }

            if (throwMe) {
              throw throwMe;
            }

            if (pending.length) {
              return { value: pending.shift() };
            }

            return (await new Promise((resolve, reject) => (deferred = { resolve, reject })))
              ? { done: true, value: undefined }
              : { value: pending.shift() };
          },
          async return() {
            dispose();
            return { done: true, value: undefined };
          },
        };
      }
      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: fetcher,
          isHeadersEditorEnabled: true,
          shouldPersistHeaders: true
        }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
`))

// Handler responsible for setting up the playground
func Handler(title string, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		err := page.Execute(w, map[string]interface{}{
			"title":              title,
			"endpoint":           endpoint,
			"endpointIsAbsolute": endpointHasScheme(endpoint),
			"version":            "3.0.1",
			"cssSRI":             "sha256-wTzfn13a+pLMB5rMeysPPR1hO7x0SwSeQI+cnw7VdbE=",
			"jsSRI":              "sha256-dLnxjV+d2rFUCtYKjbPy413/8O+Ahy7QqAhaPNlL8fk=",
			"reactSRI":           "sha256-S0lp+k7zWUMk2ixteM6HZvu8L9Eh//OVrt+ZfbCpmgY=",
			"reactDOMSRI":        "sha256-IXWO0ITNDjfnNXIu5POVfqlgYoop36bDzhodR6LW5Pc=",
		})
		if err != nil {
			panic(err)
		}
	}
}

// endpointHasScheme checks if the endpoint has a scheme.
func endpointHasScheme(endpoint string) bool {
	u, err := url.Parse(endpoint)
	return err == nil && u.Scheme != ""
}
