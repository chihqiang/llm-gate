package model

import "time"

type Log struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	RequestPath    string    `json:"request_path" gorm:"size:512;comment:请求路径"`
	RequestMethod  string    `json:"request_method" gorm:"size:16;comment:请求方法"`
	ResponseCode   int       `json:"response_code" gorm:"comment:响应状态码"`
	RequestPayload string    `json:"request_payload" gorm:"type:text;comment:请求体"`
	RequestIP      string    `json:"request_ip" gorm:"size:64;comment:请求IP"`
	RequestOS      string    `json:"request_os" gorm:"size:256;comment:操作系统"`
	RequestBrowser string    `json:"request_browser" gorm:"size:256;comment:浏览器"`
	ResponseJSON   string    `json:"response_json" gorm:"type:text;comment:响应内容"`
	ProcessTime    int64     `json:"process_time" gorm:"comment:处理耗时(ms)"`
	AccountID      int64     `json:"account_id" gorm:"comment:操作人ID"`
	AccountName    string    `json:"account_name" gorm:"size:64;comment:操作人名称"`
	Description    string    `json:"description" gorm:"size:512;comment:描述"`
	CreatedAt      time.Time `json:"created_at" gorm:"comment:创建时间"`
}

func (Log) TableName() string {
	return "sys_logs"
}
