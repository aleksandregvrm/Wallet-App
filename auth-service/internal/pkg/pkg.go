package pkg

// Inter-service communication types

type AuthenticateUserResponse struct {
	AccessToken string
	UserId      string
}

// Auth-service when assigning authenticated user a session
type UserSessionTokens struct {
	RefreshToken string
	SessionToken string
}
