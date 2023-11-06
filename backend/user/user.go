package user

// User represents a user in the system
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Database interface {
	GetUserByUsername(username string) (*User, error)
	CreateUser(user *User) error
}
