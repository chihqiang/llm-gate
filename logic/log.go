package logic

import (
	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type LogLogic struct {
	db *gorm.DB
}

func NewLogLogic(db *gorm.DB) *LogLogic {
	return &LogLogic{db: db}
}

type LogListRequest struct {
	Page             int    `form:"page" binding:"required,min=1"`
	Size             int    `form:"size" binding:"required,min=1,max=1000"`
	RequestPath      string `form:"request_path"`
	RequestIP        string `form:"request_ip"`
	RequestMethod    string `form:"request_method"`
	CurrentAccountID int64  `form:"-"`
}

type LogListResponse struct {
	Data  []model.Log `json:"data"`
	Total int64       `json:"total"`
}

func (s *LogLogic) List(req *LogListRequest) (*LogListResponse, error) {
	var logs []model.Log
	var total int64

	query := s.db.Model(&model.Log{})
	if req.CurrentAccountID > 0 {
		query = query.Where("account_id = ?", req.CurrentAccountID)
	}
	if req.RequestPath != "" {
		query = query.Where("request_path LIKE ?", "%"+req.RequestPath+"%")
	}
	if req.RequestIP != "" {
		query = query.Where("request_ip LIKE ?", "%"+req.RequestIP+"%")
	}
	if req.RequestMethod != "" {
		query = query.Where("request_method = ?", req.RequestMethod)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	return &LogListResponse{Data: logs, Total: total}, nil
}

func (s *LogLogic) Create(log *model.Log) error {
	return s.db.Create(log).Error
}
