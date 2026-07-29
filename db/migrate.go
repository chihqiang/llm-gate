package db

import (
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/hash"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Account{},
		&model.Role{},
		&model.Menu{},
		&model.Log{},
	); err != nil {
		return err
	}

	return seed(db)
}

func seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.Account{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 默认菜单
		menus := []model.Menu{
			// 仪表盘
			{PID: 0, MenuType: 1, Name: "仪表盘", Path: "/admin/dashboard", Component: "admin/dashboard/page", Icon: "LayoutDashboard", Sort: 1, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "仪表盘目录"},
			{PID: 1, MenuType: 2, Name: "数据概览", Path: "/admin/dashboard", Component: "admin/dashboard/page", Icon: "LayoutDashboard", Sort: 1, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "仪表盘页面"},
			// 系统管理目录
			{PID: 0, MenuType: 1, Name: "系统管理", Path: "/admin/sys", Component: "admin/sys/page", Icon: "Setting", Sort: 2, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "系统管理目录"},
			// 账号管理
			{PID: 3, MenuType: 2, Name: "账号管理", Path: "/admin/sys/account", Component: "admin/sys/account/page", Icon: "Users", Sort: 1, APIURL: "/api/v1/sys/accounts", APIMethod: "GET", Visible: true, Status: true, Remark: "账号管理菜单"},
			{PID: 4, MenuType: 3, Name: "账号详情", APIURL: "/api/v1/sys/accounts/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取账号详情"},
			{PID: 4, MenuType: 3, Name: "创建账号", APIURL: "/api/v1/sys/accounts", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建账号"},
			{PID: 4, MenuType: 3, Name: "更新账号", APIURL: "/api/v1/sys/accounts/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新账号"},
			{PID: 4, MenuType: 3, Name: "删除账号", APIURL: "/api/v1/sys/accounts/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除账号"},
			// 角色管理
			{PID: 3, MenuType: 2, Name: "角色管理", Path: "/admin/sys/roles", Component: "admin/sys/roles/page", Icon: "UserCog", Sort: 2, APIURL: "/api/v1/sys/roles", APIMethod: "GET", Visible: true, Status: true, Remark: "角色管理菜单"},
			{PID: 9, MenuType: 3, Name: "角色详情", APIURL: "/api/v1/sys/roles/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取角色详情"},
			{PID: 9, MenuType: 3, Name: "创建角色", APIURL: "/api/v1/sys/roles", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建角色"},
			{PID: 9, MenuType: 3, Name: "更新角色", APIURL: "/api/v1/sys/roles/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新角色"},
			{PID: 9, MenuType: 3, Name: "删除角色", APIURL: "/api/v1/sys/roles/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除角色"},
			{PID: 9, MenuType: 3, Name: "所有角色", APIURL: "/api/v1/sys/roles/all", APIMethod: "GET", Sort: 6, Visible: true, Status: true, Remark: "获取所有角色列表"},
			{PID: 9, MenuType: 3, Name: "关联菜单", APIURL: "/api/v1/sys/roles/*/menus", APIMethod: "POST", Sort: 7, Visible: true, Status: true, Remark: "关联角色和菜单"},
			// 菜单管理
			{PID: 3, MenuType: 2, Name: "菜单管理", Path: "/admin/sys/menu", Component: "admin/sys/menu/page", Icon: "ShieldUser", Sort: 3, APIURL: "/api/v1/sys/menus", APIMethod: "GET", Visible: true, Status: true, Remark: "菜单管理菜单"},
			{PID: 16, MenuType: 3, Name: "所有菜单", APIURL: "/api/v1/sys/menus/all", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取所有菜单列表"},
			{PID: 16, MenuType: 3, Name: "菜单详情", APIURL: "/api/v1/sys/menus/*", APIMethod: "GET", Sort: 3, Visible: true, Status: true, Remark: "获取菜单详情"},
			{PID: 16, MenuType: 3, Name: "创建菜单", APIURL: "/api/v1/sys/menus", APIMethod: "POST", Sort: 4, Visible: true, Status: true, Remark: "创建菜单"},
			{PID: 16, MenuType: 3, Name: "更新菜单", APIURL: "/api/v1/sys/menus/*", APIMethod: "PUT", Sort: 5, Visible: true, Status: true, Remark: "更新菜单"},
			{PID: 16, MenuType: 3, Name: "删除菜单", APIURL: "/api/v1/sys/menus/*", APIMethod: "DELETE", Sort: 6, Visible: true, Status: true, Remark: "删除菜单"},
			// 日志管理
			{PID: 3, MenuType: 2, Name: "日志管理", Path: "/admin/sys/log", Component: "admin/sys/log/page", Icon: "ScrollText", Sort: 4, APIURL: "/api/v1/sys/logs", APIMethod: "GET", Visible: true, Status: true, Remark: "日志管理菜单"},
		}
		if err := tx.Create(&menus).Error; err != nil {
			return err
		}

		// 默认角色
		role := model.Role{Name: "超级管理员", Sort: 1, Status: true, Remark: "超级管理员角色，拥有所有权限"}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		// 角色关联所有菜单
		if err := tx.Model(&role).Association("Menus").Replace(menus); err != nil {
			return err
		}

		// 默认管理员
		hashed, err := hash.BcryptHashDefault("123456")
		if err != nil {
			return err
		}
		admin := model.Account{
			Name:     "超级管理员",
			Email:    "admin@example.com",
			Password: hashed,
			Status:   true,
		}
		if err := tx.Create(&admin).Error; err != nil {
			return err
		}

		return tx.Model(&admin).Association("Roles").Replace([]model.Role{role})
	})
}
