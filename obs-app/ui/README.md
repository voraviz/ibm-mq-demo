# obs-app — UI

Vue 3 + Vite frontend for the obs-app observability demo.

This is based on `ui-app/` with the Vite proxy and dev server port updated to match `obs-app/api`.

## Ports

| Service | Port |
|---|---|
| Vite dev server | **8083** |
| Proxied API | `http://localhost:8082` |

## Start

```bash
cd obs-app/ui
npm install
npm run dev
```

Open **http://localhost:8083**

## Configuration

Copy `.env.example` to `.env`. In dev mode, leave `VITE_API_BASE_URL=` empty so the Vite proxy handles all `/api` and `/ws` requests.

```bash
cp .env.example .env
```
