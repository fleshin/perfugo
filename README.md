# Perfugo

Perfugo is an empty Go project scaffold inspired by the layout of
[fleshin/projectx](https://github.com/fleshin/projectx). It provides a minimal
entry-point, health check handler, and static Tailwind-powered landing page so
that you can start building features immediately.

## Structure

```
.
├── cmd/
│   └── server/         # Application entrypoint
├── internal/
│   ├── handlers/       # HTTP handlers and business logic
│   └── server/         # HTTP server wiring and router
├── web/
│   └── static/         # Static assets served by the file server
├── go.mod              # Module definition
└── README.md
```

## Getting started

1. Install Go 1.21 or newer.
2. Run the development server:

   ```bash
   go run ./cmd/server
   ```

3. Visit [http://localhost:8080](http://localhost:8080) to see the Tailwind
   landing page or [http://localhost:8080/healthz](http://localhost:8080/healthz)
   for the JSON health check.

The scaffold uses the Tailwind CDN instead of Bulma so you can iterate on your
front-end quickly without a build step. Update `web/static/index.html` as you
start building the UI.
