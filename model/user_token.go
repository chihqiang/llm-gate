package model

import "time"

type UserToken struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	AccountID int64      `json:"account_id" gorm:"not null;index;comment:账户ID"`
	Name      string     `json:"name" gorm:"size:64;not null;comment:令牌名称"`
	Key       string     `json:"key" gorm:"size:512;not null;comment:令牌密钥（AES 加密存储）"`
	KeyHash   string     `json:"-" gorm:"size:64;not null;uniqueIndex;comment:密钥SHA256，用于认证查询"`
	Quota     int64      `json:"quota" gorm:"default:0;comment:预算（分），0=不限"` 
	SpentCents int64     `json:"spent_cents" gorm:"default:0;comment:已消费（分）"`
	ModelIDs  string     `json:"model_ids" gorm:"size:512;comment:模型白名单 JSON 数组，空=全部"`
	Status    bool       `json:"status" gorm:"default:true;comment:状态"`
	ExpiredAt *time.Time `json:"expired_at" gorm:"comment:过期时间"`
	CreatedAt time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"comment:更新时间"`
}

func (UserToken) TableName() string {
	return "llm_user_tokens"
}
