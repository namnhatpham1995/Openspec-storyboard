# Storyboard frontend

React + TypeScript client for the local Storyboard server.

```sh
# Terminal 1, from the repository root
go run ./cmd/storyboard --project /path/to/project

# Terminal 2
cd frontend
npm run dev
```

Vite proxies `/api` requests to `http://127.0.0.1:8080` in development.

Available checks:

```sh
npm run build
npm run lint
npm test
```
