package password

import "testing"

func TestCreateHashedPassword(t *testing.T) {
	passwd := "very-strong-secret-7"
	hashedPasswd, _ := CreateHashedPassword(passwd)

	if hashedPasswd == "" {
		t.Errorf("hashed password empty")
	}

	if !CheckPassword(passwd, hashedPasswd) {
		t.Errorf("hashed password is not valid")
	}
}
