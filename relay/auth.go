package relay

import (
	"errors"
	"net/http"
	"strings"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type AuthResult struct {
	Token   *model.UserToken
	Account *model.Account
}

func Authenticate(r *http.Request, db *gorm.DB) (*AuthResult, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, errors.New("missing authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, errors.New("invalid authorization format")
	}

	key := parts[1]

	var token model.UserToken
	if err := db.Where("key = ? AND status = ?", key, true).First(&token).Error; err != nil {
		return nil, errors.New("invalid or disabled token")
	}

	if token.ExpiredAt != nil && token.ExpiredAt.Before(token.CreatedAt) {
		return nil, errors.New("token expired")
	}

	if token.Quota <= 0 {
		return nil, errors.New("insufficient quota")
	}

	var account model.Account
	if err := db.First(&account, token.AccountID).Error; err != nil {
		return nil, errors.New("account not found")
	}

	if !account.Status {
		return nil, errors.New("account disabled")
	}

	return &AuthResult{Token: &token, Account: &account}, nil
}
