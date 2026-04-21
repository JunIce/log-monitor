# Log Analysis System

A Go-based log query web server with bash extraction scripts for efficient log analysis and visualization.

## Overview

This system provides a web interface for querying and analyzing log files with the following features:

- Project-based log management
- Time range filtering for log queries
- Keyword search capabilities
- Dark theme UI
- RESTful API for log operations

## Project Structure

```
log-analysis/
├── public/           # Frontend static files
│   ├── index.html    # Main log analysis interface
│   ├── settings.html # Project configuration manager
│   └── style.css     # Shared styling
├── logs/             # Log storage directory (auto-created)
│   └── 2026-04-XX/   # Date-based subdirectories
├── main.go           # Go backend server
├── extract_log_time_range.sh # Bash extraction script
└── AGENTS.md         # Operational guidelines
```

## Setup & Installation

### Prerequisites
- Go 1.21+
- Bash (for time range extraction)
- Git (optional, for version control)

### Quick Start

1. **Run development server**
```bash
go run . -port 8888
```

2. **Build production binary**
```bash
go build -o log-server.exe . && ./log-server.exe -port 8888
```

3. **Regenerate time index** (requires bash/git bash)
```bash
bash extract_log_time_range.sh
```

## API Endpoints

### Log Types & Dates
```http
GET /api/log_types
```
**Response:**
```json
{
  "log_types": ["sys-info", "error", "access"],
  "dates": ["2026-04-19", "2026-04-20", "2026-04-21"]
}
```

### Log Query
```http
GET /api/query?log_type=sys-info&start_time=09:00:00.000&end_time=12:00:00.000&date=2026-04-19
```

### Log Content
```http
GET /api/log_content?filename=...&date=...&start_time=...&end_time=...&keyword=error
```

### Projects API
```http
GET /api/projects     # List all projects
POST /api/projects    # Add new project
PUT /api/projects     # Update project
DELETE /api/projects # Delete project
```

## Project Management

### Settings Interface

Access the project manager at `http://localhost:8888/settings.html`

**Project Configuration:**
- **Project Name**: Identifiable name for your log collection
- **Log Directory**: Path to store logs (e.g., `logs/prod`)
- **Index File**: Time range index file (defaults to `time_ranges.json`)

### Operations
- Add new projects with custom log directories
- Edit existing project configurations
- Delete projects (permanent removal)
- Persistent storage of project settings

## Log Format Requirements

### File Structure
Logs must be stored in: `logs/{YYYY-MM-DD}/{log_type}.{date}.{seq}.log`

### Timestamp Format
First token of each log line must be: `HH:mm:ss.SSS`

### Extraction Rules
- Script filters with `grep "^[0-9]"` (skips Java stack traces)
- Time range matching uses overlap logic: `entryFirst <= queryEnd && entryLast >= queryStart`

## Frontend Features

### Main Interface (`index.html`)
- File list navigation
- Log content viewer with syntax highlighting
- Time range filter controls
- Real-time keyword search
- Fullscreen/maximize functionality
- Dark theme support

### Settings Interface (`settings.html`)
- Project management dashboard
- Form validation for inputs
- Toast notifications for operations
- Responsive grid layout
- Edit/delete confirmation dialogs

## Port Configuration

- Default: 8888 (avoid 8080/8089/9999 due to Windows conflicts)
- Custom port: `go run . -port 4567`

## Development

### Code Structure
- Backend: Go server with REST API implementation
- Frontend: HTML/CSS/JS with dark theme support
- Scripts: Bash utility for log time range extraction

### Contributing
1. Fork the repository
2. Create feature branch
3. Implement changes
4. Submit pull request

## Language

- [English](README.md)
- [中文](README_CN.md)

## License

[MIT License](LICENSE) - See LICENSE file for details