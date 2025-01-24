package models

import "time"

type CommunicationWhatsappAttachment struct {
	ID                      int       `json:"id" gorm:"column:id;type:int"`
	Filename                string    `json:"filename" gorm:"column:filename;type:varchar(255)"`
	Content                 string    `json:"content" gorm:"column:content;type:text"`
	Size                    int       `json:"size" gorm:"column:size;type:int"`
	Type                    uint      `json:"type" gorm:"column:type;type:int"`
	CommunicationWhatsappID int       `json:"communicationWhatsappId" gorm:"column:communication_whatsapp_id;type:int"`
	CreatedAt               time.Time `json:"createdAt" gorm:"column:created_at;type:timestamp"`
	UpdatedAt               time.Time `json:"updatedAt" gorm:"column:updated_at;type:timestamp"`
}

func (CommunicationWhatsappAttachment) TableName() string {
	return "blasts.communication_whatsapp_attachments"
}
