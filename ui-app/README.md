# IBM MQ UI — Vue 3 + Vite Frontend

The web interface for the IBM MQ Demo. Provides a split-panel UI for putting messages onto (left panel) and getting messages from (right panel) an IBM MQ queue in real time via WebSocket.

## Prerequisites

- Node.js 20 or later
- npm 10 or later

## Configuration

The UI talks to the Quarkus API microservice. Configure the API URL via an environment variable:

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `http://localhost:8081` | Full URL of the API microservice |

In development, the Vite dev server proxies `/api` and `/ws` to `http://localhost:8081` automatically, so no configuration is needed.

For production builds, create a `.env` file (copy from [`.env.example`](.env.example)):

```bash
cp .env.example .env
# Edit .env and set VITE_API_BASE_URL to your API host
```

## Install Dependencies

```bash
cd ui-app
npm install
```

## Running in Development Mode

```bash
npm run dev
```

The UI will start on **http://localhost:8080**.

## Building for Production

```bash
npm run build
```

Static files will be output to `dist/`. Serve this directory with any static file server (nginx, `npx serve`, etc.).

## Previewing the Production Build

```bash
npm run preview
```
