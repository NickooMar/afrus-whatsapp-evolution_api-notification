package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"

	"gorm.io/gorm"
)

type WhatsappTriggerRepository struct {
	db *gorm.DB
}

type WhatsatppTriggerRepositoryInterface interface {
	GetWhatsappTriggerById(ctx context.Context, id int) (*models.WhatsappTrigger, error)
}

func NewWhatsappTriggerRepository(db *gorm.DB) *WhatsappTriggerRepository {
	return &WhatsappTriggerRepository{
		db: db,
	}
}

func (repo *WhatsappTriggerRepository) GetWhatsappTriggerById(ctx context.Context, id int) (*models.WhatsappTrigger, error) {
	var instance models.WhatsappTrigger
	result := repo.db.WithContext(ctx).Where("id = ?", id).First(&instance)
	if result.Error != nil {
		return nil, result.Error
	}
	return &instance, nil
}
