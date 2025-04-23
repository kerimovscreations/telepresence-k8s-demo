package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Todo represents a single task in the todo list
type Todo struct {
	ID        int    `json:"id"`
	Task      string `json:"task"`
	Completed bool   `json:"completed"`
}

var (
	db *sql.DB
)

// getTodosHandler handles GET requests to /todos
func getTodosHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, task, completed FROM todos")
	if err != nil {
		http.Error(w, "Failed to fetch todos", http.StatusInternalServerError)
		log.Printf("Error querying todos: %v", err)
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Task, &todo.Completed); err != nil {
			http.Error(w, "Failed to process todos", http.StatusInternalServerError)
			log.Printf("Error scanning todo row: %v", err)
			return
		}
		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error processing todos", http.StatusInternalServerError)
		log.Printf("Error after scanning todos: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todos); err != nil {
		http.Error(w, "Failed to encode todos", http.StatusInternalServerError)
		log.Printf("Error encoding todos: %v", err)
	}
}

// createTodoHandler handles POST requests to /todos
func createTodoHandler(w http.ResponseWriter, r *http.Request) {
	var newTodo Todo
	if err := json.NewDecoder(r.Body).Decode(&newTodo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body: %v", err)
		return
	}

	// Insert the new todo into the database
	err := db.QueryRow(
		"INSERT INTO todos (task, completed) VALUES ($1, $2) RETURNING id",
		newTodo.Task + " (intercepted locally)", newTodo.Completed,
	).Scan(&newTodo.ID)

	if err != nil {
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		log.Printf("Error inserting todo: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newTodo); err != nil {
		http.Error(w, "Failed to encode created todo", http.StatusInternalServerError)
		log.Printf("Error encoding created todo: %v", err)
	}
	log.Printf("Created todo: %+v", newTodo)
}

// initDB initializes the database connection
func initDB() (*sql.DB, error) {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")  // Empty default, forces proper configuration
	dbname := getEnv("POSTGRES_DBNAME", "postgres")

	// Connection string for local PostgreSQL (port 5432)
	connStr := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Check if we have environment variables for DB connection
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		connStr = dbURL
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to PostgreSQL")
	return db, nil
}

// Helper function to get environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// initSchema ensures the database schema is set up
func initSchema(db *sql.DB) error {
	// Create the todos table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			task VARCHAR(255) NOT NULL,
			completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed);
	`)
	return err
}

func main() {
	var err error
	// Initialize database connection
	db, err = initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	if err = initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Add a sample todo if the table is empty
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count); err == nil && count == 0 {
		_, err = db.Exec("INSERT INTO todos (task, completed) VALUES ($1, $2)", "Learn Go", false)
		if err != nil {
			log.Printf("Failed to insert sample todo: %v", err)
		} else {
			log.Println("Added sample todo")
		}
	}

	http.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getTodosHandler(w, r)
		case http.MethodPost:
			createTodoHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	go func() {
		port80 := ":8080"
		log.Printf("Server starting on port %s", port80)
		if err := http.ListenAndServe(port80, nil); err != nil {
			log.Fatalf("Could not start server: %s", err)
		}
	}()

	go func() {
		port81 := ":8081"
		log.Printf("Server starting on port %s", port81)
		if err := http.ListenAndServe(port81, nil); err != nil {
			log.Fatalf("Could not start server: %s", err)
		}
	}()

	port82 := ":8082"
	log.Printf("Server starting on port %s", port82)
	if err := http.ListenAndServe(port82, nil); err != nil {
		log.Fatalf("Could not start server: %s", err)
	}
}
