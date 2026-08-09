# Gospelfast

Fast full-text search Bible web app in Go. HTMX + Alpine.js + TailwindCSS frontend, PostgreSQL full-text search backend.

## Features

- **Full-text search** — PostgreSQL `tsvector` with `<mark>` highlighting
- **Multiple translations** — Bible texts, commentaries, and general books via SWORD import
- **SWORD rawzip import** — paste a CrossWire URL, import from admin dashboard with live progress
- **Bible reader** — book/chapter navigation with direct chapter jump and copy-to-clipboard
- **Inline commentary** — click any verse number to load commentary (e.g. KingComments)
- **Comparison** — view two translations side by side
- **Verse of the Day** — deterministic daily verse from KJV on the home page
- **User accounts** — login/logout, session-based auth, admin/reader roles
- **Admin dashboard** — manage translations, import texts, add/delete users
- **Redis caching** — optional read-through cache for chapters and search results
- **Swagger API docs** — `/swagger/index.html` with all endpoints
- **Dark mode** — toggle persists in localStorage, works on all pages
- **Mobile responsive** — hamburger menu on small screens

## Quick Start

```bash
git clone https://github.com/gospelfast/gospelfast.git && cd gospelfast

# Start PostgreSQL + Redis (optional)
sudo docker compose up -d

# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
make migrate-up

# Import KJV from bible-api.com (~30 min)
go run ./cmd/gospelfast-cli/ seed

# Start server (auto-seeds admin user on first run)
go run ./cmd/gospelfast/
# → http://localhost:8080
# → Login: http://localhost:8080/login (admin / gospelfast)
# → Admin: http://localhost:8080/admin
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable` | PostgreSQL connection |
| `REDIS_URL` | `localhost:6379` | Redis (optional) |
| `PORT` | `8080` | HTTP port |
| `ADMIN_PASSWORD` | `gospelfast` | Admin password (set before first startup) |

## CLI

```bash
# Import from SWORD rawzip
go run ./cmd/gospelfast-cli/ import \
  --source https://www.crosswire.org/.../KJVA.zip --name KJVA --format rawzip

# Import local files
go run ./cmd/gospelfast-cli/ import --source bible.xml --name ESV --format osis

# Download KJV from bible-api.com
go run ./cmd/gospelfast-cli/ seed
```

## API

Interactive docs at `/swagger/index.html`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/translations` | List translations |
| `GET` | `/api/books?t=KJV` | List books |
| `GET` | `/api/verses?ref=John+3:16&t=KJV` | Get verse |
| `GET` | `/api/chapters/KJV/1?book=Gen` | Get chapter |
| `GET` | `/api/search?q=love&t=KJV` | Full-text search |
| `GET` | `/api/commentary?t=KINGCOMMENTS&book=Gen&ref=Gen.1.1` | Commentary entry |
| `GET` | `/api/genbooks?t=ENOCH` | Browse genbooks |

### Admin (session auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin` | Dashboard |
| `POST` | `/admin/api/imports` | Start import |
| `POST` | `/admin/api/users` | Create user |
| `DELETE` | `/admin/api/users/{id}` | Delete user |

## Architecture

```
cmd/{gospelfast,gospelfast-cli}/   # Server + CLI
internal/
  bible/    # Models, reference parser
  db/       # PostgreSQL pool, migrations, queries
  import/   # OSIS/VPL/SWORD parsers + pipeline
  api/      # REST handlers
  admin/    # Auth, job manager, dashboard handlers
  web/      # Page handlers + templates
  cache/    # Redis read-through
  seed/     # KJV API downloader
web/templates/   # Go HTML (HTMX + Alpine + Tailwind)
```

## License

MIT
