package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Database interface defines the methods for database operations
type Database interface {
	GetTemplates() ([]Template, error)
	AddTemplate(prompt string) error
	DeleteTemplate(id int) error
	GetUserByUsername(username string) (*User, error)
	CreateUser(user *User) error
}

// User represents a user in the system
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Template represents a template in the system
type Template struct {
	ID     int    `json:"id"`
	Prompt string `json:"prompt"`
}

// SQLiteDB implements the Database interface for SQLite database
type SQLiteDB struct {
	db *sql.DB
}

const jwtSecret = "your-secret-key"

func NewSQLiteDB(db *sql.DB) *SQLiteDB {
	return &SQLiteDB{db: db}
}

// GetTemplates fetches all templates from the database.
func (s *SQLiteDB) GetTemplates() ([]Template, error) {
	rows, err := s.db.Query("SELECT id, prompt FROM templates")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var template Template
		err := rows.Scan(&template.ID, &template.Prompt)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// AddTemplate adds a new template to the database.
func (s *SQLiteDB) AddTemplate(prompt string) error {
	_, err := s.db.Exec("INSERT INTO templates (prompt) VALUES (?)", prompt)
	return err
}

// DeleteTemplate deletes a template from the database by ID.
func (s *SQLiteDB) DeleteTemplate(id int) error {
	_, err := s.db.Exec("DELETE FROM templates WHERE id=?", id)
	return err
}

// GetUserByUsername retrieves a user by their username from the database.
func (s *SQLiteDB) GetUserByUsername(username string) (*User, error) {
	row := s.db.QueryRow("SELECT id, username, password FROM users WHERE username=?", username)
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		// User not found
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user in the database.
func (s *SQLiteDB) CreateUser(user *User) error {
	// Check if the username already exists
	existingUser, err := s.GetUserByUsername(user.Username)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return errors.New("username already exists")
	}

	// Hash the user's password before storing it in the database
	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user.Username, hashedPassword)
	return err
}

// GenerateToken generates a JWT token for the given user ID.
func GenerateToken(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
	})
	return token.SignedString([]byte(jwtSecret))
}

// VerifyToken verifies the authenticity of a JWT token.
func VerifyToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
}

// HashPassword hashes the given password using bcrypt.
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync() // Flushes buffer, if any

	// Initialize SQLite database
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		logger.Error("Error opening database", zap.Error(err))
		return
	}
	defer db.Close()

	// Create tables if they do not exist
	_, err = db.Exec(`
																																																																																																																CREATE TABLE IF NOT EXISTS users (
																																																																																																																			id INTEGER PRIMARY KEY AUTOINCREMENT,
																																																																																																																						username TEXT UNIQUE NOT NULL,
																																																																																																																									password TEXT NOT NULL
																																																																																																																											)
																																																																																																																												`)
	if err != nil {
		logger.Error("Error creating users table", zap.Error(err))
		return
	}

	_, err = db.Exec(`
																																																																																																																																					CREATE TABLE IF NOT EXISTS templates (
																																																																																																																																								id INTEGER PRIMARY KEY AUTOINCREMENT,
																																																																																																																																											prompt TEXT NOT NULL
																																																																																																																																													)
																																																																																																																																														`)
	if err != nil {
		logger.Error("Error creating templates table", zap.Error(err))
		return
	}

	// Create a new SQLiteDB instance
	database := NewSQLiteDB(db)

	// Initialize HTTP router
	r := mux.NewRouter()

	// Register routes with handlers
	r.HandleFunc("/register", RegisterHandler(database)).Methods("POST")
	r.HandleFunc("/login", LoginHandler(database)).Methods("POST")
	r.Handle("/templates", AuthMiddleware(GetTemplatesHandler(database))).Methods("GET")
	r.Handle("/templates", AuthMiddleware(AddTemplateHandler(database))).Methods("POST")
	r.Handle("/templates/{id}", AuthMiddleware(DeleteTemplateHandler(database))).Methods("DELETE")

	// Start the server
	port := ":8080"
	logger.Info("Server started on port " + port)
	err = http.ListenAndServe(port, r)
	if err != nil {
		logger.Error("Error starting server", zap.Error(err))
	}
}

// RegisterHandler handles user registration.
func RegisterHandler(db Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		err = db.CreateUser(&user)
		if err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			return
		}

		token, err := GenerateToken(user.ID)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

// LoginHandler handles user login.
func LoginHandler(db Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		storedUser, err := db.GetUserByUsername(user.Username)
		if err != nil {
			http.Error(w, "Error fetching user", http.StatusInternalServerError)
			return
		}

		if storedUser == nil || bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password)) != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(storedUser.ID)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

// AuthMiddleware is a middleware that authenticates incoming requests.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := VerifyToken(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID := int(claims["user_id"].(float64))
			// You can use userID for further authentication or authorization checks
			// For example: user := getUserByID(userID)
			// if user == nil {
			//   http.Error(w, "Unauthorized", http.StatusUnauthorized)
			//   return
			// }
			// r = r.WithContext(context.WithValue(r.Context(), "user", user))
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}

// GetTemplatesHandler handles fetching templates.
func GetTemplatesHandler(db Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		templates, err := db.GetTemplates()
		if err != nil {
			http.Error(w, "Error fetching templates", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
	}
}

// AddTemplateHandler handles adding a new template.
func AddTemplateHandler(db Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request map[string]string
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		prompt, ok := request["prompt"]
		if !ok {
			http.Error(w, "Prompt not provided", http.StatusBadRequest)
			return
		}

		err = db.AddTemplate(prompt)
		if err != nil {
			http.Error(w, "Error adding template", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteTemplateHandler handles deleting a template by ID.
func DeleteTemplateHandler(db Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		templateID, ok := params["id"]
		if !ok {
			http.Error(w, "Invalid template ID", http.StatusBadRequest)
			return
		}

		id, err := fmt.Sprint(templateID)
		if err != nil {
			http.Error(w, "Invalid template ID", http.StatusBadRequest)
			return
		}

		err = db.DeleteTemplate(id)
		if err != nil {
			http.Error(w, "Error deleting template", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
