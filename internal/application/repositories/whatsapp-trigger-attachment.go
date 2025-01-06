package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"

	"gorm.io/gorm"
)

type WhatsappTriggerAttachmentRepository struct {
	db *gorm.DB
}

type WhatsatppTriggerAttachmentRepositoryInterface interface {
	GetByTriggerId(ctx context.Context, triggerId int) ([]models.WhatsappTriggerAttachment, error)
}

func NewWhatsappTriggerAttachmentRepository(db *gorm.DB) *WhatsappTriggerAttachmentRepository {
	return &WhatsappTriggerAttachmentRepository{
		db: db,
	}
}

func (repo *WhatsappTriggerAttachmentRepository) GetByTriggerId(ctx context.Context, triggerId uint) ([]models.WhatsappTriggerAttachment, error) {
	var attachments []models.WhatsappTriggerAttachment
	result := repo.db.WithContext(ctx).Where("whatsapp_trigger_id = ?", triggerId).Find(&attachments)
	if result.Error != nil {
		return nil, result.Error
	}
	return attachments, nil
}
