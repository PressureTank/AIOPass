package main

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"github.com/neo-cypher/AIOPass/database/sqlite"
	"github.com/neo-cypher/AIOPass/template"
	"github.com/neo-cypher/AIOPass/user"
)

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

	// Create instances of database, template, and user packages
	dbInstance := sqlite.NewSQLiteDB(db)
	templateHandler := template.NewTemplateHandler(dbInstance)
	userHandler := user.NewUserHandler(dbInstance)

	// Initialize HTTP router
	r := mux.NewRouter()

	// Register routes with handlers
	r.HandleFunc("/register", userHandler.RegisterHandler).Methods("POST")
	r.HandleFunc("/login", userHandler.LoginHandler).Methods("POST")
	r.Handle("/templates", template.AuthMiddleware(templateHandler.GetTemplatesHandler)).Methods("GET")
	r.Handle("/templates", template.AuthMiddleware(templateHandler.AddTemplateHandler)).Methods("POST")
	r.Handle("/templates/{id}", template.AuthMiddleware(templateHandler.DeleteTemplateHandler)).Methods("DELETE")

	// Start the server
	port := ":8080"
	logger.Info("Server started on port " + port)
	err = http.ListenAndServe(port, r)
	if err != nil {
		logger.Error("Error starting server", zap.Error(err))
	}
}
