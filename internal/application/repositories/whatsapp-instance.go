package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type WhatsappInstanceRepository struct {
	db *gorm.DB
}

type WhatsappInstanceRepositoryInterface interface {
	Update(ctx context.Context, instance *models.WhatsappInstance) error
	GetWhatsappInstanceById(ctx context.Context, id int) (*models.WhatsappInstance, error)
	GetWhatsappInstancesByOrganization(ctx context.Context, whatsappInstance *models.WhatsappInstance) ([]models.WhatsappInstance, error)
}

func NewWhatsappInstanceRepository(db *gorm.DB) *WhatsappInstanceRepository {
	return &WhatsappInstanceRepository{
		db: db,
	}
}

func (repo *WhatsappInstanceRepository) Update(ctx context.Context, instance *models.WhatsappInstance) error {
	result := repo.db.WithContext(ctx).Model(&instance).Updates(instance)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (repo *WhatsappInstanceRepository) GetWhatsappInstanceById(ctx context.Context, id int) (*models.WhatsappInstance, error) {
	var instance models.WhatsappInstance
	result := repo.db.WithContext(ctx).Where("id = ?", id).First(&instance)
	if result.Error != nil {
		return nil, result.Error
	}
	return &instance, nil
}

func (repo *WhatsappInstanceRepository) GetWhatsappInstancesByOrganization(ctx context.Context, whatsappInstance *models.WhatsappInstance) ([]models.WhatsappInstance, error) {
	var instances []models.WhatsappInstance
	result := repo.db.WithContext(ctx).Where("organization_id = ? AND id != ?", whatsappInstance.OrganizationID, whatsappInstance.ID).Order("created_at ASC").Find(&instances)
	if result.Error != nil {
		return nil, result.Error
	}

	fmt.Printf("Whatsapp instances: %v\n", instances)
	return instances, nil
}
