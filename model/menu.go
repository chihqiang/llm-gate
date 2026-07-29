package model

import "time"

type Menu struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	PID       int64      `json:"pid" gorm:"default:0;comment:父级ID"`
	MenuType  int        `json:"menu_type" gorm:"default:1;comment:菜单类型 1=目录 2=菜单 3=按钮"`
	Name      string     `json:"name" gorm:"size:64;not null;comment:菜单名称" binding:"required"`
	Path      string     `json:"path" gorm:"size:256;comment:路由路径"`
	Component string     `json:"component" gorm:"size:256;comment:组件路径"`
	Icon      string     `json:"icon" gorm:"size:64;comment:图标"`
	Sort      int        `json:"sort" gorm:"default:0;comment:排序"`
	APIURL    string     `json:"api_url" gorm:"size:256;comment:接口地址"`
	APIMethod string     `json:"api_method" gorm:"size:16;comment:请求方法"`
	Visible   bool       `json:"visible" gorm:"default:true;comment:是否可见"`
	Status    bool       `json:"status" gorm:"default:true;comment:状态"`
	Remark    string     `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt *time.Time `json:"-" gorm:"index;comment:删除时间"`
}

func (Menu) TableName() string {
	return "sys_menus"
}
