package models

type WhatsappSentEvent struct {
	ID             int    `json:"id" gorm:"column:id;type:int"`
	OrganizationID int    `json:"organizationId" gorm:"column:organization_id;type:int"`
	LeadID         int    `json:"leadId" gorm:"column:lead_id;type:int"`
	PhoneNumber    string `json:"phoneNumber" gorm:"column:phone_number;type:varchar(255)"`
	ExternalID     string `json:"externalId" gorm:"column:external_id;type:varchar(255)"`
	ExternalTable  string `json:"externalTable" gorm:"column:external_table;type:varchar(255)"`
	MessageID      string `json:"messageId" gorm:"column:message_id;type:varchar(255)"`
	EventType      int    `json:"eventType" gorm:"column:event_type;type:int"`
	DateEvent      string `json:"dateEvent" gorm:"column:date_event;type:varchar(255)"`
	Event          JSONB  `gorm:"type:jsonb" json:"event"`
}

func (WhatsappSentEvent) TableName() string {
	return "whatsapp.sent"
}
