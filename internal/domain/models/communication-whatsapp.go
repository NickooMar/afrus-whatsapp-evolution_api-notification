package models

import "time"

type CommunicationWhatsapp struct {
	ID              int                               `json:"id" gorm:"column:id;type:int"`
	TemplateID      int                               `json:"templateId" gorm:"column:template_id;type:int"`
	CreatedAt       time.Time                         `json:"createdAt" gorm:"column:created_at;type:timestamp"`
	UpdatedAt       time.Time                         `json:"updatedAt" gorm:"column:updated_at;type:timestamp"`
	UserID          int                               `json:"userId" gorm:"column:user_id;type:int"`
	CommunicationID int                               `json:"communicationId" gorm:"column:communication_id;type:int"`
	OrganizationID  int                               `json:"organizationId" gorm:"column:organization_id;type:int"`
	Content         string                            `json:"content" gorm:"column:content;type:text"`
	Instances       []CommunicationWhatsappInstance   `gorm:"foreignKey:CommunicationWhatsappID" json:"instances"`
	Attachments     []CommunicationWhatsappAttachment `gorm:"foreignKey:CommunicationWhatsappID" json:"attachments"`
}

func (CommunicationWhatsapp) TableName() string {
	return "blasts.communication_whatsapps"
}
