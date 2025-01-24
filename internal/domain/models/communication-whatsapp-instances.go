package models

import "time"

type CommunicationWhatsappInstance struct {
	ID                      int              `json:"id" gorm:"column:id;type:int"`
	CommunicationWhatsappID int              `json:"communicationWhatsappId" gorm:"column:communication_whatsapp_id;type:int"`
	WhatsappInstanceID      int              `json:"whatsappInstanceId" gorm:"column:whatsapp_instance_id;type:int"`
	CreatedAt               time.Time        `json:"createdAt" gorm:"column:created_at;type:timestamp"`
	UpdatedAt               time.Time        `json:"updatedAt" gorm:"column:updated_at;type:timestamp"`
	WhatsappInstance        WhatsappInstance `gorm:"foreignKey:WhatsappInstanceID" json:"whatsappInstance"`
}

func (CommunicationWhatsappInstance) TableName() string {
	return "blasts.communication_whatsapp_instances"
}
