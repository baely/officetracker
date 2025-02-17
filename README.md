# Office Tracker

---

[Live demo](https://iwasintheoffice.com)

![Screenshot of web app](docs/assets/screenshot.png)

## Description

Office Tracker is a web application designed to track daily office presence on a monthly basis. The application supports two modes of operation:

1. **Integrated Mode**: Stores data on a cloud postgres instance and uses GitHub OAuth for user authentication.
2. **Standalone Mode**: Employs local storage (SQLite) for data and does not require user authentication.

## Architecture

### Overview
The application follows a microservice architecture pattern with the following key components:

### Components
1. **Frontend**
   - Single-page application built with vanilla JavaScript
   - Server-side rendered templates using Go's html/template
   - Responsive design for both desktop and mobile views
   - Real-time updates using form submissions

2. **Backend Application (Go)**
   - HTTP server handling API requests and serving web pages
   - Authentication system using GitHub OAuth
   - JWT-based session management
   - Report generation (CSV, PDF formats)
   - RESTful API endpoints for attendance tracking
   - Developer API for integrations

3. **Data Storage**
   - **Primary Database**: PostgreSQL (hosted on Supabase)
     - Stores user accounts and attendance records
     - Links GitHub accounts to user profiles
     - Simple data model with users, attendance entries, and GitHub associations
     - Automated SQL migrations for schema management
   
   - **Cache Layer**: Redis (hosted on Redis Cloud)
     - Stores temporary OAuth states for GitHub authentication
     - Short-lived session data
     - No rate limiting implementation
     - Both services operate within free tier limits

4. **Deployment**
   - **Application**: Google Cloud Run
     - Containerized deployment
     - Auto-scaling based on demand
     - Zero downtime updates
   - **Domain**: Custom domain with SSL/TLS
   - **CI/CD**: GitHub Actions pipeline
     - Automated testing
     - Docker image building
     - Deployment to Cloud Run

### Data Flow
1. User accesses the application through HTTPS
2. Authentication:
   - For new users: GitHub OAuth flow with state management in Redis
   - For existing users: JWT-based authentication
3. User actions:
   - Attendance records stored directly in PostgreSQL
   - OAuth states temporarily cached in Redis
   - Reports generated directly from PostgreSQL data
   
### Development Setup
The development environment uses Docker Compose to simulate the production setup:
- Local PostgreSQL instance
- Local Redis instance
- Automated database migrations
- Hot-reloading for rapid development

### Modes of Operation
1. **Integrated Mode** (Production):
   - Full cloud infrastructure
   - Multi-user support
   - GitHub authentication
   - Complete feature set

2. **Standalone Mode**:
   - SQLite database for local storage
   - No authentication required
   - Perfect for single-user or offline use
   - Minimal dependencies

## Run Guide

### Integrated Mode

To run the application in Integrated Mode, it is recommended to use the provided Docker Compose setup. This will start:
- A Postgres database
- A database migration service
- The main application

You will need to:
1. Create a `config/local.env` file based on the provided `sample.env`
2. Configure the following environment variables:
   - `APP_*`: Basic application settings
   - `DOMAIN_*`: Domain configuration
   - `POSTGRES_*`: Database connection details
   - `REDIS_*`: Redis configuration
   - `GITHUB_*`: GitHub OAuth credentials
   - `SIGNING_KEY`: JWT signing key

**Running the Application:**

```shell
docker compose up
```

Alternatively, you can pull the latest Docker image from the Google Artifact Registry:

`asia-southeast1-docker.pkg.dev/baileybutler-syd/officetracker/officetracker:latest`

### Standalone Mode

To run the application in Standalone Mode, compile the Go code with the `standalone` build tag:

```shell
go build -tags=standalone -o officetracker .
```

Then, execute the binary:

```shell
./officetracker
```

You can also run the binary with optional flags:
- `-port`: Specify the port the server should listen on (default is `8080`).
- `-database`: Specify the path to the SQLite database file (default is `officetracker.db`).

Example:

```shell
./officetracker -port 1234 -database mydb.db
