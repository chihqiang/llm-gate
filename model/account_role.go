package model

type AccountRole struct {
	AccountID int64 `json:"account_id" gorm:"primaryKey;comment:账号ID"`
	RoleID    int64 `json:"role_id" gorm:"primaryKey;comment:角色ID"`
}

func (AccountRole) TableName() string {
	return "sys_account_roles"
}
