package jwt

type Config struct {
	Secret         string
	ExpiresMinutes int64
}
