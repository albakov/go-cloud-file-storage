package user

type Session struct {
	Id           int64
	UserId       int64
	RefreshToken string
	ExpiredAt    string
}
