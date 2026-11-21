package model

import "time"

type Provider struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name      string     `json:"name" gorm:"size:64;not null;comment:服务商名称"`
	BaseURL   string     `json:"base_url" gorm:"size:256;not null;comment:接口地址"`
	APIKey    string     `json:"-" gorm:"size:512;not null;comment:API密钥"`
	Status    bool       `json:"status" gorm:"default:true;comment:状态"`
	Priority  int        `json:"priority" gorm:"default:0;comment:优先级"`
	Weight    int        `json:"weight" gorm:"default:1;comment:权重"`
	Remark    string     `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt *time.Time `json:"-" gorm:"index;comment:删除时间"`
}

func (Provider) TableName() string {
	return "llm_providers"
}
