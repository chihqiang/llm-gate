package relay

import "chihqiang/llm-gate/model"

type AuthResult struct {
	Token   *model.UserToken
	Account *model.Account
}
