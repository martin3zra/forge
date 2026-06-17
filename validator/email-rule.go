package validator

import (
	"net/mail"
)

type emailRule struct{}

func newEmailRule() emailRule {
	return emailRule{}
}

func (emailRule) validEmailAddress(address string) bool {
	_, err := mail.ParseAddress(address)
	return err == nil
}
