package model

type RoleMenu struct {
	RoleID int64 `json:"role_id" gorm:"primaryKey;comment:角色ID"`
	MenuID int64 `json:"menu_id" gorm:"primaryKey;comment:菜单ID"`
}

func (RoleMenu) TableName() string {
	return "sys_role_menus"
}
