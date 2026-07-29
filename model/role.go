package model

import "time"

type Role struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name      string     `json:"name" gorm:"size:64;not null;comment:角色名称" binding:"required"`
	Sort      int        `json:"sort" gorm:"default:0;comment:排序"`
	Status    bool       `json:"status" gorm:"default:true;comment:状态"`
	Remark    string     `json:"remark" gorm:"size:512;comment:备注"`
	Menus     []Menu     `json:"menus" gorm:"many2many:sys_role_menus;comment:菜单关联"`
	CreatedAt time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt *time.Time `json:"-" gorm:"index;comment:删除时间"`
}

func (Role) TableName() string {
	return "sys_roles"
}
