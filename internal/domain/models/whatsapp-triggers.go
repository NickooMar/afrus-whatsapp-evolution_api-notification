package models

import (
	"time"

	"gorm.io/gorm"
)

type WhatsappTrigger struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"column:name" json:"name"`
	InternalName    string         `gorm:"column:internal_name" json:"internal_name"`
	Phone           string         `gorm:"column:phone" json:"phone"`
	Status          int            `gorm:"column:status" json:"status"`
	Type            int            `gorm:"column:type" json:"type"`
	All             bool           `gorm:"column:all" json:"all"`
	Content         string         `gorm:"column:content" json:"content"`
	TemplateEventID *uint          `gorm:"column:template_event_id" json:"template_event_id"`
	CampaignID      *uint          `gorm:"column:campaign_id" json:"campaign_id"`
	FormID          *uint          `gorm:"column:form_id" json:"form_id"`
	OrganizationID  uint           `gorm:"column:organization_id" json:"organization_id"`
	LanguageCode    string         `gorm:"column:language_code" json:"language_code"`
	GASource        *string        `gorm:"column:ga_source" json:"ga_source"`
	GAMedium        *string        `gorm:"column:ga_medium" json:"ga_medium"`
	GAName          *string        `gorm:"column:ga_name" json:"ga_name"`
	GAContent       *string        `gorm:"column:ga_content" json:"ga_content"`
	CreatedAt       time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (WhatsappTrigger) TableName() string {
	return "autoresponders.whatsapp_triggers"
}
