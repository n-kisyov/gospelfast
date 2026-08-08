# Gospelfast

Fast full-text search Bible web application written in Go.

## Features

- **Full-text search** across all imported Bible texts with PostgreSQL `tsvector` + highlighting
- **Multiple translation support** — import KJV, ESV, or any SWORD-format module
- **SWORD rawzip import** — paste a CrossWire rawzip URL and import directly from the admin dashboard
- **General book support** — import and browse GenBook modules (e.g., Book of Enoch)
- **Bible reader** — browse by book/chapter with direct chapter navigation
- **Translation comparison** — view two translations side by side
- **Redis caching** — optional read-through cache for chapter text and search results
- **Admin dashboard** — manage translations, import new texts with progress tracking
- **Swagger API docs** — interactive API documentation at `/swagger/index.html`
- **Dark mode** — toggle with sun/moon icon (persists in localStorage)
- **Mobile responsive** — works on phones and tablets

## Quick Start

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Redis 7+ (optional, app runs without it)
- SWORD utilities (`mod2imp`) for rawzip import: `sudo apt install libsword-utils`
- `golang-migrate` CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

### Setup

```bash
# Clone
git clone https://github.com/gospelfast/gospelfast.git
cd gospelfast

# Start database
sudo docker compose up -d

# Run migrations
make migrate-up

# Import KJV from bible-api.com (~30 min)
go run ./cmd/gospelfast-cli/ seed

# Or import a SWORD rawzip module
go run ./cmd/gospelfast-cli/ import \
  --source https://www.crosswire.org/ftpmirror/pub/sword/packages/rawzip/KJVA.zip \
  --name KJVA --format rawzip

# Start server
go run ./cmd/gospelfast/
```

Open http://localhost:8080

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable` | PostgreSQL connection |
| `REDIS_URL` | `localhost:6379` | Redis address (optional) |
| `PORT` | `8080` | HTTP listen port |
| `ADMIN_PASSWORD` | `gospelfast` | Admin dashboard password |
| `TEMPLATES_DIR` | `web/templates` | HTML template directory |
| `STATIC_DIR` | `web/static` | Static files directory |

## Architecture

```
cmd/
  gospelfast/          # Web server
  gospelfast-cli/      # CLI import tool

internal/
  bible/               # Core domain: Verse, Book, Translation models, reference parser
  db/                  # PostgreSQL: connection pool, migrations, queries
  import/              # Import pipeline
    osis/              #   OSIS XML parser (container + milestone verse formats)
    vpl/               #   VPL verse-per-line parser
    sword/             #   SWORD rawzip converter (mod2imp wrapper)
  api/                 # REST API handlers
  admin/               # Admin dashboard: auth, job manager, import UI
  cache/               # Redis read-through caching
  search/              # Full-text search query builder
  seed/                # KJV downloader from bible-api.com
  web/                 # Frontend page handlers

web/templates/         # Go HTML templates (HTMX + Alpine.js + TailwindCSS)
```

## API

Interactive docs at `/swagger/index.html` when the server is running.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/translations` | List all translations |
| `GET` | `/api/books?t=KJV` | List books for a translation |
| `GET` | `/api/verses?ref=John+3:16&t=KJV` | Get verse by reference |
| `GET` | `/api/chapters/KJV/1?book=Gen` | Get chapter verses |
| `GET` | `/api/search?q=love&t=KJV` | Full-text search |
| `GET` | `/api/genbooks?t=ENOCH` | Browse genbook entries |

### Admin (Basic Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin` | Dashboard |
| `POST` | `/admin/api/imports` | Start import job |
| `GET` | `/admin/api/imports/:id` | Import job status |
| `DELETE` | `/admin/api/translations/:id` | Delete translation |

## License

MIT
