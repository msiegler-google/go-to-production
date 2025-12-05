# Milestone 0: Baseline Application (Local Development)

This document outlines how to run the simple, local version of the app (without any cloud dependencies).

## 1. Checkout this Milestone

To run the local version of the app, you **must** check out the `baseline` tag. The `main` branch contains cloud-specific code (Secret Manager, etc.) that will not run locally without GCP credentials.

```bash
git checkout tags/baseline
```

## 2. What was Implemented?

This is the starting point of our journey: a "Toy App" designed to be simple to understand but lacking production features.

**Key Features:**
*   **Go Backend**: A simple HTTP server using `net/http`.
*   **PostgreSQL**: Persistent storage for To-Do items.
*   **Docker Compose**: Orchestrates the app and database locally.
*   **Frontend**: Basic HTML/JS served statically.

**Benefits:**
*   **Simplicity**: Easy to run on a laptop with just Docker.
*   **Fast Feedback**: No cloud deployment time; changes are instant.

## 3. Pitfalls & Considerations

*   **No Security**: Database passwords are in plain text in `.env` files.
*   **Single Point of Failure**: If your laptop dies, the app dies. No high availability.
*   **No Observability**: Logs are just printed to stdout. No metrics or tracing.

## 4. Alternatives Considered

*   **SQLite**: Would be even simpler (no separate DB container).
    *   *Why Postgres?* To mimic a real production stack where the DB is a separate service, allowing us to demonstrate Cloud SQL migration later.

## Usage Instructions

### 1. Create a `.env` file
Create a file named `.env` in the root directory:
```
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=todoapp_db
DATABASE_URL=postgres://user:password@db:5432/todoapp_db?sslmode=disable
```

### 2. Build and Run with Docker Compose
```bash
docker-compose up --build
```
The app will be available at [http://localhost:8080](http://localhost:8080).

### 3. API Endpoints

*   **`GET /todos`**: Retrieve all to-do items.
*   **`POST /todos`**: Add a new to-do item.
    *   Request Body: `{"task": "New task description"}`
*   **`PUT /todos/{id}`**: Update a to-do item.
    *   Request Body: `{"completed": true}`
*   **`DELETE /todos/{id}`**: Delete a to-do item.
