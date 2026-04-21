# Log Analysis

Go log query web server with bash extraction script.

## Commands

```bash
# Run server (dev mode)
go run . -port 8888

# Build (production)
go build -o log-server.exe . && ./log-server.exe -port 8888

# Regenerate time index (requires bash/git bash)
bash extract_log_time_range.sh
```

## Port: 8888

Avoid 8080/8089/9999 (Windows conflicts).

## API

- `/api/log_types` → `{log_types: [...], dates: [...]}`
- `/api/query?log_type=sys-info&start_time=09:00:00.000&end_time=12:00:00.000&date=2026-04-19`
- `/api/log_content?filename=...&date=...&start_time=...&end_time=...&keyword=error`

## Time Range Matching

Overlap matching: `entryFirst <= queryEnd && entryLast >= queryStart`. See `main.go:171`.

## Log Format

- Files: `{log_type}.{date}.{seq}.log` in `logs/2026-04-XX/`
- Timestamps: `HH:mm:ss.SSS` (first token)
- Extract script filters with `grep "^[0-9]"` (skips Java stack traces)

## Frontend

- Static in `public/`: `index.html` (dark theme), `style.css`
- Features: file list, log viewer, time filter, keyword search, maximize
