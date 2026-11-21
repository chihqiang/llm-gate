package model

import "time"

type ModelConfig struct {
	ID                int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name              string     `json:"name" gorm:"size:128;not null;index:idx_llm_model_name;comment:模型名称"`
	ProviderID        int64      `json:"provider_id" gorm:"not null;uniqueIndex:idx_llm_provider_upstream;comment:服务商ID"`
	UpstreamModelName string     `json:"upstream_model_name" gorm:"size:128;not null;uniqueIndex:idx_llm_provider_upstream;comment:上游模型名"`
	ModelRatio        float64    `json:"model_ratio" gorm:"type:decimal(10,4);default:1.0;comment:模型倍率"`
	CompletionRatio   float64    `json:"completion_ratio" gorm:"type:decimal(10,4);default:1.0;comment:补全倍率"`
	Weight            int        `json:"weight" gorm:"default:1;comment:权重（同模型多服务商时按权重分发）"`
	Status            bool       `json:"status" gorm:"default:true;comment:状态"`
	Remark            string     `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt         time.Time  `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt         time.Time  `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt         *time.Time `json:"-" gorm:"index;comment:删除时间"`
}

func (ModelConfig) TableName() string {
	return "llm_models"
}
