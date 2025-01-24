package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &j)
}

type WhatsappInstance struct {
	ID             uint                            `gorm:"primaryKey" json:"id"`
	InstanceName   string                          `gorm:"column:instanceName" json:"instanceName"`
	InstanceID     string                          `gorm:"column:instanceId" json:"instanceId"`
	Owner          string                          `gorm:"column:owner" json:"owner"`
	Data           JSONB                           `gorm:"type:jsonb" json:"data"`
	OrganizationID uint                            `gorm:"column:organization_id" json:"organizationId"`
	CreatedAt      time.Time                       `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time                       `gorm:"column:updated_at" json:"updatedAt"`
	Communications []CommunicationWhatsappInstance `gorm:"foreignKey:WhatsappInstanceID" json:"communications"`
}

func (WhatsappInstance) TableName() string {
	return "whatsapp_instances"
}
