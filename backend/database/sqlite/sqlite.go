package sqlite

import (
	"database/sql"
	"errors"

<<<<<<< HEAD
	_ "github.com/PressureTank/AIOPass/template"
	_ "github.com/PressureTank/AIOPass/user"
=======
	_ "github.com/PressureTank/AIOPass/backend/template"
	_ "github.com/PressureTank/AIOPass/backend/user"
>>>>>>> 9df1370 ((chore)packages: implements packages for database, template, and user ops.)
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type SQLiteDB struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewSQLiteDB(db *sql.DB, logger *zap.Logger) *SQLiteDB {
	return &SQLiteDB{
		db:     db,
		logger: logger,
	}
}

func (s *SQLiteDB) GetTemplates() ([]Template, error) {
	rows, err := s.db.Query("SELECT id, prompt FROM templates")
	if err != nil {
		s.logger.Error("Error fetching templates from database", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var template Template
		err := rows.Scan(&template.ID, &template.Prompt)
		if err != nil {
			s.logger.Error("Error scanning template row", zap.Error(err))
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

func (s *SQLiteDB) AddTemplate(prompt string) error {
	_, err := s.db.Exec("INSERT INTO templates (prompt) VALUES (?)", prompt)
	if err != nil {
		s.logger.Error("Error inserting template into database", zap.Error(err))
		return err
	}
	return nil
}

func (s *SQLiteDB) DeleteTemplate(id int) error {
	_, err := s.db.Exec("DELETE FROM templates WHERE id=?", id)
	if err != nil {
		s.logger.Error("Error deleting template from database", zap.Error(err))
		return err
	}
	return nil
}

func (s *SQLiteDB) GetUserByUsername(username string) (*User, error) {
	row := s.db.QueryRow("SELECT id, username, password FROM users WHERE username=?", username)
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		// User not found
		return nil, nil
	} else if err != nil {
		s.logger.Error("Error fetching user from database", zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (s *SQLiteDB) CreateUser(user *User) error {
	// Check if the username already exists
	existingUser, err := s.GetUserByUsername(user.Username)
	if err != nil {
		s.logger.Error("Error checking existing user", zap.Error(err))
		return err
	}
	if existingUser != nil {
		return errors.New("username already exists")
	}

	// Hash the user's password before storing it in the database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Error hashing user password", zap.Error(err))
		return err
	}

	_, err = s.db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user.Username, hashedPassword)
	if err != nil {
		s.logger.Error("Error inserting user into database", zap.Error(err))
		return err
	}
	return nil
}
