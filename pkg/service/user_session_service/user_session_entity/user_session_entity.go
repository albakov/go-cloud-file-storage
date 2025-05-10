package user_session_entity

type UserSession struct {
	UserId       int64
	RefreshToken string
	ExpiredAt    string
}
