package storage

import "fmt"

var ErrNotFound = fmt.Errorf("sql: no rows in result set")
var ErrDuplicateNotAllowed = fmt.Errorf("sql: duplicate not allowed")
