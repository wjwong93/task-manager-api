package db

// schema.go provides data models in DB
import (
	"time"
)

// Task corresponds to a row in `tasks` table
type Task struct {
	ID        uint64    `db:"id"`
	Title     string    `db:"title"`
	Priority  string    `db:"priority"`
	DueDate   time.Time    `db:"due_date"`
	IsDone    bool      `db:"is_done"`
	Description string  `db:"description"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type User struct {
	ID       uint64 `db:"id"`
	Name     string `db:"name"`
	Password []byte `db:"password"`
}

type TaskOwner struct {
	UserID   uint64 `db:"user_id"`
	TaskID   uint64 `db:"task_id"`
}