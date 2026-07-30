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
		&model.Provider{},
		&model.ModelConfig{},
		&model.UserToken{},
		&model.UsageLog{},
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
		var allMenus []model.Menu

		createMenu := func(m *model.Menu) error {
			if err := tx.Create(m).Error; err != nil {
				return err
			}
			allMenus = append(allMenus, *m)
			return nil
		}

		createChild := func(parent *model.Menu, children ...model.Menu) error {
			for i := range children {
				children[i].PID = parent.ID
				if err := createMenu(&children[i]); err != nil {
					return err
				}
			}
			return nil
		}

		dashDir := &model.Menu{PID: 0, MenuType: 1, Name: "仪表盘", Path: "/admin/dashboard", Component: "admin/dashboard/page", Icon: "LayoutDashboard", Sort: 1, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "仪表盘目录"}
		if err := createMenu(dashDir); err != nil {
			return err
		}
		if err := createChild(dashDir,
			model.Menu{MenuType: 2, Name: "数据概览", Path: "/admin/dashboard", Component: "admin/dashboard/page", Icon: "LayoutDashboard", Sort: 1, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "仪表盘页面"},
			model.Menu{MenuType: 3, Name: "仪表盘统计", APIURL: "/api/v1/dashboard/stats", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取仪表盘统计"},
			model.Menu{MenuType: 3, Name: "用量列表", APIURL: "/api/v1/llm/usage", APIMethod: "GET", Sort: 3, Visible: true, Status: true, Remark: "获取用量列表"},
			model.Menu{MenuType: 3, Name: "用量统计", APIURL: "/api/v1/llm/usage/stats", APIMethod: "GET", Sort: 4, Visible: true, Status: true, Remark: "获取用量统计"},
		); err != nil {
			return err
		}

		sysDir := &model.Menu{PID: 0, MenuType: 1, Name: "系统管理", Path: "/admin/sys", Component: "admin/sys/page", Icon: "Settings", Sort: 2, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "系统管理目录"}
		if err := createMenu(sysDir); err != nil {
			return err
		}
		if err := createChild(sysDir,
			model.Menu{MenuType: 2, Name: "账号管理", Path: "/admin/sys/account", Component: "admin/sys/account/page", Icon: "Users", Sort: 1, APIURL: "/api/v1/sys/accounts", APIMethod: "GET", Visible: true, Status: true, Remark: "账号管理菜单"},
			model.Menu{MenuType: 3, Name: "账号详情", APIURL: "/api/v1/sys/accounts/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取账号详情"},
			model.Menu{MenuType: 3, Name: "创建账号", APIURL: "/api/v1/sys/accounts", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建账号"},
			model.Menu{MenuType: 3, Name: "更新账号", APIURL: "/api/v1/sys/accounts/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新账号"},
			model.Menu{MenuType: 3, Name: "删除账号", APIURL: "/api/v1/sys/accounts/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除账号"},
		); err != nil {
			return err
		}
		if err := createChild(sysDir,
			model.Menu{MenuType: 2, Name: "角色管理", Path: "/admin/sys/roles", Component: "admin/sys/roles/page", Icon: "UserCog", Sort: 2, APIURL: "/api/v1/sys/roles", APIMethod: "GET", Visible: true, Status: true, Remark: "角色管理菜单"},
			model.Menu{MenuType: 3, Name: "角色详情", APIURL: "/api/v1/sys/roles/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取角色详情"},
			model.Menu{MenuType: 3, Name: "创建角色", APIURL: "/api/v1/sys/roles", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建角色"},
			model.Menu{MenuType: 3, Name: "更新角色", APIURL: "/api/v1/sys/roles/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新角色"},
			model.Menu{MenuType: 3, Name: "删除角色", APIURL: "/api/v1/sys/roles/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除角色"},
			model.Menu{MenuType: 3, Name: "所有角色", APIURL: "/api/v1/sys/roles/all", APIMethod: "GET", Sort: 6, Visible: true, Status: true, Remark: "获取所有角色列表"},
			model.Menu{MenuType: 3, Name: "关联菜单", APIURL: "/api/v1/sys/roles/*/menus", APIMethod: "POST", Sort: 7, Visible: true, Status: true, Remark: "关联角色和菜单"},
		); err != nil {
			return err
		}
		if err := createChild(sysDir,
			model.Menu{MenuType: 2, Name: "菜单管理", Path: "/admin/sys/menu", Component: "admin/sys/menu/page", Icon: "ShieldUser", Sort: 3, APIURL: "/api/v1/sys/menus", APIMethod: "GET", Visible: true, Status: true, Remark: "菜单管理菜单"},
			model.Menu{MenuType: 3, Name: "所有菜单", APIURL: "/api/v1/sys/menus/all", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取所有菜单列表"},
			model.Menu{MenuType: 3, Name: "菜单详情", APIURL: "/api/v1/sys/menus/*", APIMethod: "GET", Sort: 3, Visible: true, Status: true, Remark: "获取菜单详情"},
			model.Menu{MenuType: 3, Name: "创建菜单", APIURL: "/api/v1/sys/menus", APIMethod: "POST", Sort: 4, Visible: true, Status: true, Remark: "创建菜单"},
			model.Menu{MenuType: 3, Name: "更新菜单", APIURL: "/api/v1/sys/menus/*", APIMethod: "PUT", Sort: 5, Visible: true, Status: true, Remark: "更新菜单"},
			model.Menu{MenuType: 3, Name: "删除菜单", APIURL: "/api/v1/sys/menus/*", APIMethod: "DELETE", Sort: 6, Visible: true, Status: true, Remark: "删除菜单"},
		); err != nil {
			return err
		}
		if err := createChild(sysDir,
			model.Menu{MenuType: 2, Name: "日志管理", Path: "/admin/sys/log", Component: "admin/sys/log/page", Icon: "ScrollText", Sort: 4, APIURL: "/api/v1/sys/logs", APIMethod: "GET", Visible: true, Status: true, Remark: "日志管理菜单"},
		); err != nil {
			return err
		}

		llmDir := &model.Menu{PID: 0, MenuType: 1, Name: "LLM 网关", Path: "/admin/sys/llm", Component: "admin/sys/llm/page", Icon: "Cpu", Sort: 3, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "LLM 网关目录"}
		if err := createMenu(llmDir); err != nil {
			return err
		}
		if err := createChild(llmDir,
			model.Menu{MenuType: 2, Name: "服务商管理", Path: "/admin/sys/llm/providers", Component: "admin/sys/llm/providers/page", Icon: "Server", Sort: 1, APIURL: "/api/v1/llm/providers", APIMethod: "GET", Visible: true, Status: true, Remark: "服务商管理菜单"},
			model.Menu{MenuType: 3, Name: "所有服务商", APIURL: "/api/v1/llm/providers/all", APIMethod: "GET", Sort: 1, Visible: true, Status: true, Remark: "获取所有服务商列表"},
			model.Menu{MenuType: 3, Name: "服务商详情", APIURL: "/api/v1/llm/providers/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取服务商详情"},
			model.Menu{MenuType: 3, Name: "创建服务商", APIURL: "/api/v1/llm/providers", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建服务商"},
			model.Menu{MenuType: 3, Name: "更新服务商", APIURL: "/api/v1/llm/providers/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新服务商"},
			model.Menu{MenuType: 3, Name: "删除服务商", APIURL: "/api/v1/llm/providers/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除服务商"},
			model.Menu{MenuType: 3, Name: "同步预览", APIURL: "/api/v1/llm/providers/*/sync-models/preview", APIMethod: "GET", Sort: 6, Visible: true, Status: true, Remark: "预览上游模型"},
			model.Menu{MenuType: 3, Name: "同步模型", APIURL: "/api/v1/llm/providers/*/sync-models", APIMethod: "POST", Sort: 7, Visible: true, Status: true, Remark: "同步上游模型"},
		); err != nil {
			return err
		}
		if err := createChild(llmDir,
			model.Menu{MenuType: 2, Name: "模型管理", Path: "/admin/sys/llm/models", Component: "admin/sys/llm/models/page", Icon: "Cpu", Sort: 2, APIURL: "/api/v1/llm/models", APIMethod: "GET", Visible: true, Status: true, Remark: "模型管理菜单"},
			model.Menu{MenuType: 3, Name: "所有模型", APIURL: "/api/v1/llm/models/all", APIMethod: "GET", Sort: 1, Visible: true, Status: true, Remark: "获取所有模型列表"},
			model.Menu{MenuType: 3, Name: "模型详情", APIURL: "/api/v1/llm/models/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取模型详情"},
			model.Menu{MenuType: 3, Name: "创建模型", APIURL: "/api/v1/llm/models", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建模型"},
			model.Menu{MenuType: 3, Name: "更新模型", APIURL: "/api/v1/llm/models/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新模型"},
			model.Menu{MenuType: 3, Name: "删除模型", APIURL: "/api/v1/llm/models/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除模型"},
		); err != nil {
			return err
		}
		if err := createChild(llmDir,
			model.Menu{MenuType: 2, Name: "API Key", Path: "/admin/sys/llm/tokens", Component: "admin/sys/llm/tokens/page", Icon: "Key", Sort: 3, APIURL: "/api/v1/llm/tokens", APIMethod: "GET", Visible: true, Status: true, Remark: "API Key 管理菜单"},
			model.Menu{MenuType: 3, Name: "Token详情", APIURL: "/api/v1/llm/tokens/*", APIMethod: "GET", Sort: 2, Visible: true, Status: true, Remark: "获取Token详情"},
			model.Menu{MenuType: 3, Name: "创建Token", APIURL: "/api/v1/llm/tokens", APIMethod: "POST", Sort: 3, Visible: true, Status: true, Remark: "创建Token"},
			model.Menu{MenuType: 3, Name: "更新Token", APIURL: "/api/v1/llm/tokens/*", APIMethod: "PUT", Sort: 4, Visible: true, Status: true, Remark: "更新Token"},
			model.Menu{MenuType: 3, Name: "删除Token", APIURL: "/api/v1/llm/tokens/*", APIMethod: "DELETE", Sort: 5, Visible: true, Status: true, Remark: "删除Token"},
			model.Menu{MenuType: 3, Name: "Token密钥", APIURL: "/api/v1/llm/tokens/*/reveal", APIMethod: "GET", Sort: 6, Visible: true, Status: true, Remark: "查看Token完整密钥"},
		); err != nil {
			return err
		}
		if err := createChild(llmDir,
			model.Menu{MenuType: 2, Name: "聊天", Path: "/admin/sys/llm/chat", Component: "admin/sys/llm/chat/page", Icon: "MessageSquare", Sort: 4, APIURL: "", APIMethod: "*", Visible: true, Status: true, Remark: "聊天"},
		); err != nil {
			return err
		}

		role := model.Role{Name: "超级管理员", Sort: 1, Status: true, Remark: "超级管理员角色，拥有所有权限"}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		if err := tx.Model(&role).Association("Menus").Replace(allMenus); err != nil {
			return err
		}

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
