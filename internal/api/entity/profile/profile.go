package profile

type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret"`
} // @name LoginRequest

type RegisterRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret"`
} // @name RegisterRequest

type LoginResponse struct {
	AccessToken string `json:"access_token" example:"secret-access-token"`
} // @name LoginResponse

type ProfileResponse struct {
	Email string `json:"email" example:"user@example.com"`
} // @name ProfileResponse

type RefreshAccessTokenResponse struct {
	AccessToken string `json:"access_token" example:"secret-access-token"`
} // @name RefreshAccessTokenResponse
