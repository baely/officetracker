# Redesign

- [Redesign](#redesign)
    - [Motivation and goals](#motivation-and-goals)
    - [Current state](#current-state)

## Motivation and goals

The current implementation has been iteratively designed around the
web-client <--> backend interaction.

I want to implement a set of developer APIs which requires intentional design to
avoid introducing complexities.

Broadly speaking, I want to achieve the following:

- Separation of concerns
- Meaningful API endpoints
- API versioning
- Documentation
- Consistent UX

And some stretch goals:

- Testing
- Observability

## Current state

**note**: auth and standalone are mutually exclusive.

| endpoint                      | method | auth | standalaone | desc                                          |
|-------------------------------|--------|------|-------------|-----------------------------------------------|
| /                             | GET    | No   | Yes         | Redirects to /form                            |
| /login                        | GET    | No   | No          | Login page                                    |
| /form                         | GET    | JWT  | Yes         | Form page. Redirects to /form/{current-month} |
| /form/{month}                 | GET    | JWT  | Yes         | Form page                                     |
| /user-state/{month}           | GET    | JWT  | Yes         | API endpoint for retrieving state             |
| /submit                       | POST   | JWT  | Yes         | API endpoint for submitting form              |
| /setup                        | GET    | JWT  | No          | Setup page                                    |
| /download                     | GET    | JWT  | Yes         | Download page                                 |
| /static/github-mark-white.png | GET    | No   | No          | Static file                                   |
| /auth                         | GET    | No   | No          | GitHub callback                               |

## Proposed state

| endpoint                           | method | auth       | standalone | desc                                     |
|------------------------------------|--------|------------|------------|------------------------------------------|
| /                                  | GET    | No         | No         | Hero                                     |
| /                                  | GET    | JWT        | Yes        | Form page. Redirects to /{current-month} |
| /{month}                           | GET    | JWT        | Yes        | Form page                                |
| /login                             | GET    | No         | No         | Login page                               |
| /auth                              | GET    | No         | No         | GitHub callback                          |
| /privacy                           | GET    | No         | No         | Privacy policy                           |
| /tos                               | GET    | No         | No         | Terms of service                         |
| /api/v1/state/{year}               | GET    | JWT/Secret | Yes        | Get state for year                       |
| /api/v1/state/{year}/{month}       | GET    | JWT/Secret | Yes        | Get state for month                      |
| /api/v1/state/{year}/{month}       | POST   | JWT/Secret | Yes        | Set state for month                      |
| /api/v1/state/{year}/{month}/{day} | GET    | JWT/Secret | Yes        | Get state for date                       |
| /api/v1/state/{year}/{month}/{day} | POST   | JWT/Secret | Yes        | Set state for date                       |
| /api/v1/developer/secret           | POST   | JWT        | No         | Get developer client secret              |
| /api/v1/health/ping                | GET    | No         | Yes        | Health check                             |
| /api/v1/health/auth                | GET    | Secret     | No         | Check auth flow                          |                           
