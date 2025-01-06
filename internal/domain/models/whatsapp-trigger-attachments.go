package models

import (
	"time"
)

type WhatsappTriggerAttachment struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Filename          string    `gorm:"column:filename" json:"filename"`
	Content           string    `gorm:"column:content" json:"content"`
	Size              int64     `gorm:"column:size" json:"size"`
	Type              uint      `gorm:"column:type" json:"type"`
	WhatsappTriggerID uint      `gorm:"column:whatsapp_trigger_id" json:"whatsapp_trigger_id"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (WhatsappTriggerAttachment) TableName() string {
	return "autoresponders.whatsapp_trigger_attachments"
}
