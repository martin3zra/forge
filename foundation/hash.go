package foundation

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

type Hash struct{}

func NewHashable() Hash {
	return Hash{}
}
func (Hash) Make(value string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(value), 14)
	if err != nil {
		panic(err.Error())
	}

	return string(bytes)
}

func (Hash) Check(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (Hash) Sha1HMAC(input, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

func (Hash) Sha1(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

func (h Hash) HMACEquals(raw, signature, secret string) bool {
	expected := h.Sha1HMAC(raw, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (h Hash) Sha1Equals(raw, expectedHash string) bool {
	computedHash := h.Sha1(raw)
	return computedHash == expectedHash
}
