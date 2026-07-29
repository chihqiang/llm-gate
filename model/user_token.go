package model

import "time"

type UserToken struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	AccountID int64      `json:"account_id" gorm:"not null;index;comment:账户ID"`
	Name      string     `json:"name" gorm:"size:64;not null;comment:令牌名称"`
	Key       string     `json:"key" gorm:"size:128;not null;uniqueIndex;comment:令牌密钥"`
	Quota     int64      `json:"quota" gorm:"default:0;comment:额度"`
	Status    bool       `json:"status" gorm:"default:true;comment:状态"`
	ExpiredAt *time.Time `json:"expired_at" gorm:"comment:过期时间"`
	CreatedAt time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"comment:更新时间"`
}

func (UserToken) TableName() string {
	return "llm_user_tokens"
}
