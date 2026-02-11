package authentication

type AuthenticationService interface {
	Token() (string, error)
}
