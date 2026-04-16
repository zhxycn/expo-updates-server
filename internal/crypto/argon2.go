package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2 struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewArgon2(time, memory uint32, threads uint8, keyLen uint32) *Argon2 {
	return &Argon2{
		time:    time,
		memory:  memory,
		threads: threads,
		keyLen:  keyLen,
	}
}

func DefaultArgon2() *Argon2 {
	return NewArgon2(1, 64*1024, 4, 32)
}

func (a *Argon2) Hash(plain []byte) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(plain, salt, a.time, a.memory, a.threads, a.keyLen)

	var b strings.Builder
	b.WriteString("$argon2id$v=")
	b.WriteString(strconv.Itoa(argon2.Version))
	b.WriteString("$m=")
	b.WriteString(strconv.FormatUint(uint64(a.memory), 10))
	b.WriteString(",t=")
	b.WriteString(strconv.FormatUint(uint64(a.time), 10))
	b.WriteString(",p=")
	b.WriteString(strconv.FormatUint(uint64(a.threads), 10))
	b.WriteString("$")
	b.WriteString(base64.RawStdEncoding.EncodeToString(salt))
	b.WriteString("$")
	b.WriteString(base64.RawStdEncoding.EncodeToString(hash))

	return b.String(), nil
}

func (a *Argon2) Verify(encoded string, plain []byte) bool {
	// $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.SplitN(encoded, "$", 7)
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	params := parts[3]
	var memory, time uint32
	var threads uint8

	for _, segment := range strings.Split(params, ",") {
		key, val, ok := strings.Cut(segment, "=")
		if !ok {
			return false
		}

		n, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return false
		}

		switch key {
		case "m":
			memory = uint32(n)
		case "t":
			time = uint32(n)
		case "p":
			threads = uint8(n)
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	computed := argon2.IDKey(plain, salt, time, memory, threads, uint32(len(expected)))

	return subtle.ConstantTimeCompare(computed, expected) == 1
}
