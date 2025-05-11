package user

import "database/sql"

type User struct {
	Id       int64
	Email    sql.NullString
	Password string
}
