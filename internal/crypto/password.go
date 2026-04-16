package crypto

type Password interface {
	Hash(password string) (string, error)
	Check(encoded, password string) bool
}

type password struct {
	argon2 *Argon2
}

func NewPassword(argon2 *Argon2) Password {
	return &password{
		argon2: argon2,
	}
}

func (p password) Hash(password string) (string, error) {
	return p.argon2.Hash([]byte(password))
}

func (p password) Check(encoded, password string) bool {
	return p.argon2.Verify(encoded, []byte(password))
}
