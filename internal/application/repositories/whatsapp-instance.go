package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"

	"gorm.io/gorm"
)

type WhatsappInstanceRepository struct {
	db *gorm.DB
}

type WhatsappInstanceRepositoryInterface interface {
	GetWhatsappInstanceById(ctx context.Context, id int) (*models.WhatsappInstance, error)
	Update(ctx context.Context, instance *models.WhatsappInstance) error
}

func NewWhatsappInstanceRepository(db *gorm.DB) *WhatsappInstanceRepository {
	return &WhatsappInstanceRepository{
		db: db,
	}
}

func (repo *WhatsappInstanceRepository) GetWhatsappInstanceById(ctx context.Context, id int) (*models.WhatsappInstance, error) {
	var instance models.WhatsappInstance
	result := repo.db.WithContext(ctx).Where("id = ?", id).First(&instance)
	if result.Error != nil {
		return nil, result.Error
	}
	return &instance, nil
}

func (repo *WhatsappInstanceRepository) Update(ctx context.Context, instance *models.WhatsappInstance) error {
	result := repo.db.WithContext(ctx).Model(&instance).Updates(instance)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
