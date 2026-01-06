package auth

type AuthService interface {
	Token() (string, error)
}
