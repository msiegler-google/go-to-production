// Written by Gemini CLI
// This file is licensed under the MIT License.
// See the LICENSE file for details.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/cenkalti/backoff/v4"
	_ "github.com/lib/pq"
	"github.com/sony/gobreaker"
)

// ... (metrics definitions remain same) ...

// Todo represents a single todo item.
type Todo struct {
	ID        int    `json:"id"`
	Task      string `json:"task"`
	Completed bool   `json:"completed"`
}

// DBConfig holds database connection parameters.
type DBConfig struct {
	DBUser     string `json:"db_user"`
	DBName     string `json:"db_name"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBReadHost string `json:"db_read_host"`
	DBReadPort string `json:"db_read_port"`
}

var (
	db     *sql.DB
	dbRead *sql.DB
)

var cb *gobreaker.CircuitBreaker

func init() {
	var st gobreaker.Settings
	st.Name = "DatabaseCB"
	st.MaxRequests = 1            // Requests allowed in half-open state
	st.Interval = 0               // Cyclic period of closed state (0 = never clear counts)
	st.Timeout = 30 * time.Second // Duration of open state
	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	}
	st.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
		slog.Warn("Circuit Breaker state changed", "name", name, "from", from, "to", to)
	}

	cb = gobreaker.NewCircuitBreaker(st)
}

// Helper for retrying operations with Circuit Breaker
func executeWithResilience(op func() error) error {
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, retryOperation(op)
	})
	return err
}

// Helper for retrying operations (internal)
func retryOperation(op func() error) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 100 * time.Millisecond
	b.MaxInterval = 2 * time.Second
	b.MaxElapsedTime = 5 * time.Second // Fail fast for user requests

	return backoff.RetryNotify(op, b, func(err error, d time.Duration) {
		slog.Warn("Database operation failed, retrying...", "error", err, "duration", d)
	})
}

// ... (main function remains mostly same, but initDB changes) ...

func initDB(config DBConfig) {
	var err error

	dbUser := config.DBUser
	dbName := config.DBName
	dbHost := config.DBHost
	dbPort := config.DBPort

	// Primary Connection
	connStr := fmt.Sprintf("postgres://%s:dummy-password@%s:%s/%s?sslmode=disable", dbUser, dbHost, dbPort, dbName)
	slog.Info("Connecting to PRIMARY database", "url", connStr)

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 2 * time.Minute

	op := func() error {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return err
		}
		return db.Ping()
	}

	err = backoff.RetryNotify(op, b, func(err error, d time.Duration) {
		slog.Warn("Could not connect to PRIMARY database, retrying...", "error", err, "duration", d)
	})

	if err != nil {
		slog.Error("Could not connect to the PRIMARY database", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to PRIMARY database")

	// Read Replica Connection
	if config.DBReadHost != "" {
		dbReadHost := config.DBReadHost
		dbReadPort := config.DBReadPort
		if dbReadPort == "" {
			dbReadPort = dbPort
		}

		readConnStr := fmt.Sprintf("postgres://%s:dummy-password@%s:%s/%s?sslmode=disable", dbUser, dbReadHost, dbReadPort, dbName)
		slog.Info("Connecting to READ REPLICA", "url", readConnStr)

		opRead := func() error {
			dbRead, err = sql.Open("postgres", readConnStr)
			if err != nil {
				return err
			}
			return dbRead.Ping()
		}

		// We can be more lenient with Read Replica connection failure
		err = backoff.RetryNotify(opRead, b, func(err error, d time.Duration) {
			slog.Warn("Could not connect to READ REPLICA, retrying...", "error", err, "duration", d)
		})

		if err != nil {
			slog.Error("Could not connect to READ REPLICA, falling back to PRIMARY", "error", err)
			dbRead = db // Fallback to primary
		} else {
			slog.Info("Successfully connected to READ REPLICA")
		}
	} else {
		slog.Info("No Read Replica configured, using PRIMARY for reads")
		dbRead = db
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "Database connection not initialized", http.StatusInternalServerError)
		return
	}
	if err := db.Ping(); err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Check Read Replica too if distinct
	if dbRead != db && dbRead != nil {
		if err := dbRead.Ping(); err != nil {
			slog.Warn("Read Replica ping failed", "error", err)
			// Don't fail health check if only read replica is down?
			// Or maybe we should? For now, let's just log it.
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ... (serveIndex remains same) ...

// ... (handleTodos remains same) ...

// ... (handleTodo remains same) ...

func getTodos(w http.ResponseWriter, r *http.Request) {
	var todos []Todo

	err := executeWithResilience(func() error {
		// Use dbRead for SELECT
		rows, err := dbRead.Query("SELECT id, task, completed FROM todos ORDER BY id")
		if err != nil {
			return err
		}
		defer rows.Close()

		todos = []Todo{} // Reset slice on retry
		for rows.Next() {
			var t Todo
			if err := rows.Scan(&t.ID, &t.Task, &t.Completed); err != nil {
				return err
			}
			todos = append(todos, t)
		}
		return rows.Err()
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			http.Error(w, "Service Unavailable (Circuit Breaker Open)", http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func addTodo(w http.ResponseWriter, r *http.Request) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := executeWithResilience(func() error {
		return db.QueryRow("INSERT INTO todos (task) VALUES ($1) RETURNING id, completed", t.Task).Scan(&t.ID, &t.Completed)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			http.Error(w, "Service Unavailable (Circuit Breaker Open)", http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func updateTodo(w http.ResponseWriter, r *http.Request, id int) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := executeWithResilience(func() error {
		_, err := db.Exec("UPDATE todos SET completed = $1 WHERE id = $2", t.Completed, id)
		return err
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			http.Error(w, "Service Unavailable (Circuit Breaker Open)", http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	err := executeWithResilience(func() error {
		_, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
		return err
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			http.Error(w, "Service Unavailable (Circuit Breaker Open)", http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ... (accessSecretVersion remains same) ...

func accessSecretVersion(name string) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}
