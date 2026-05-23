# ArmTrade

A full-stack stock market analysis web application featuring real-time price data, AI-powered analysis via Google Gemini, interactive charts, and portfolio tracking.

## Tech Stack

- **Frontend:** Angular 18 · PrimeNG · lightweight-charts (TradingView) · jsPDF
- **Backend:** Go 1.25 · Gin · JWT (HS256) · gorilla/websocket
- **Database:** MongoDB 7
- **Infrastructure:** Docker Compose · Nginx reverse proxy

## Features

- Stock search, candlestick charts, fundamentals, news, and dividends (Yahoo Finance)
- AI-powered analysis, screening, comparison, and earnings summaries (Google Gemini)
- Real-time price updates via WebSocket
- User authentication with JWT
- Watchlist and portfolio management
- PDF export of analyses
- Market overview with sector heatmap, top movers, and macro indicators

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

## Getting Started

1. **Clone the repository**

   ```bash
   git clone <repo-url> && cd ArmTrade
   ```

2. **Configure environment variables**

   Create `backend/.env`:

   ```env
   JWT_SECRET=your-secret-key
   GEMINI_API_KEY=your-gemini-api-key   # optional — AI features degrade gracefully
   ```

3. **Start the application**

   ```bash
   docker compose up --build
   ```

   This starts three containers:

   | Service    | Port  | Description                     |
   |------------|-------|---------------------------------|
   | `frontend` | 80    | Nginx serving Angular SPA       |
   | `backend`  | 8080  | Go API server                   |
   | `mongo`    | 27017 | MongoDB 7 with persistent volume|

4. **Open the app** at [http://localhost](http://localhost)

## Project Structure

```
├── backend/
│   ├── api/          # HTTP handlers, middleware, WebSocket
│   ├── config/       # Environment config
│   ├── db/           # MongoDB connection
│   ├── models/       # Data models (User, Watchlist, Analysis)
│   └── services/     # Yahoo Finance & Gemini API clients
├── frontend/
│   └── src/app/
│       ├── pages/        # Angular page components
│       ├── services/     # HTTP & WebSocket services
│       └── interceptors/ # Auth interceptor
├── diagrams/         # draw.io architecture diagrams
└── docker-compose.yml
```

## Environment Variables

| Variable       | Required | Default                 | Description              |
|----------------|----------|-------------------------|--------------------------|
| `JWT_SECRET`   | Yes      | —                       | Secret for JWT signing   |
| `GEMINI_API_KEY`| No      | —                       | Google Gemini API key    |
| `CORS_ORIGIN`  | No       | `http://localhost:4200` | Allowed CORS origin      |
| `PORT`         | No       | `8080`                  | Backend listen port      |
| `MONGO_URI`    | No       | `mongodb://mongo:27017` | MongoDB connection URI   |

## Development

**Backend** (requires Go 1.25+):

```bash
cd backend
cp .env.example .env   # configure secrets
go run main.go
```

**Frontend** (requires Node 20+):

```bash
cd frontend
npm install
ng serve
```

The dev server runs at `http://localhost:4200` and proxies API calls to the backend.
