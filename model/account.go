package model

import "time"

type Account struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name      string     `json:"name" gorm:"size:64;not null;comment:用户名" binding:"required"`
	Email     string     `json:"email" gorm:"size:128;uniqueIndex;not null;comment:邮箱" binding:"required,email"`
	Password  string     `json:"-" gorm:"size:256;not null;comment:密码"`
	Status    bool       `json:"status" gorm:"default:true;comment:状态"`
	Roles     []Role     `json:"roles" gorm:"many2many:sys_account_roles;comment:角色关联"`
	CreatedAt time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt *time.Time `json:"-" gorm:"index;comment:删除时间"`
}

func (Account) TableName() string {
	return "sys_accounts"
}
