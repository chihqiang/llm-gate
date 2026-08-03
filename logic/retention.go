package logic

import (
	"context"
	"time"

	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

// RetentionCleaner 按保留策略周期清理过期数据：用量日志、余额流水、过期 Token。
type RetentionCleaner struct {
	db  *gorm.DB
	cfg config.RetentionConfig
}

func NewRetentionCleaner(db *gorm.DB, cfg config.RetentionConfig) *RetentionCleaner {
	return &RetentionCleaner{db: db, cfg: cfg}
}

func (c *RetentionCleaner) Start(ctx context.Context) {
	if c.cfg.CheckInterval <= 0 {
		c.cfg.CheckInterval = 24 * time.Hour
	}
	interval := c.cfg.CheckInterval
	go func() {
		// 启动后先执行一次，再按周期执行
		c.clean(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.clean(ctx)
			}
		}
	}()
}

func (c *RetentionCleaner) clean(ctx context.Context) {
	now := time.Now()

	if c.cfg.UsageDays > 0 {
		cutoff := now.AddDate(0, 0, -c.cfg.UsageDays)
		res := c.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&model.UsageLog{})
		if res.Error != nil {
			logger.GetGlobal().Errorf("清理用量日志失败: %v", res.Error)
		} else if res.RowsAffected > 0 {
			logger.GetGlobal().Infof("清理过期用量日志 %d 条", res.RowsAffected)
		}
	}

	if c.cfg.TransactionDays > 0 {
		cutoff := now.AddDate(0, 0, -c.cfg.TransactionDays)
		res := c.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&model.Transaction{})
		if res.Error != nil {
			logger.GetGlobal().Errorf("清理余额流水失败: %v", res.Error)
		} else if res.RowsAffected > 0 {
			logger.GetGlobal().Infof("清理过期余额流水 %d 条", res.RowsAffected)
		}
	}

	if c.cfg.ExpiredTokenGraceDays > 0 {
		// 清理已过期且超过宽限期的 Token
		cutoff := now.AddDate(0, 0, -c.cfg.ExpiredTokenGraceDays)
		res := c.db.WithContext(ctx).Where("expired_at IS NOT NULL AND expired_at < ?", cutoff).Delete(&model.UserToken{})
		if res.Error != nil {
			logger.GetGlobal().Errorf("清理过期 Token 失败: %v", res.Error)
		} else if res.RowsAffected > 0 {
			logger.GetGlobal().Infof("清理过期 Token %d 条", res.RowsAffected)
		}
	}
}
