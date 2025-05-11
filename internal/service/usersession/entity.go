package usersession

type UserSession struct {
	UserId       int64
	RefreshToken string
	ExpiredAt    string
}
