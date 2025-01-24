package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"

	"gorm.io/gorm"
)

type CommunicationWhatsappRepository struct {
	DB *gorm.DB
}

type CommunicationWhatsappRepositoryInterface interface {
	FindById(ctx context.Context, id int) (*models.CommunicationWhatsapp, error)
}

func NewCommunicationWhatsappRepository(db *gorm.DB) *CommunicationWhatsappRepository {
	return &CommunicationWhatsappRepository{DB: db}
}

func (repo *CommunicationWhatsappRepository) FindById(ctx context.Context, id int) (*models.CommunicationWhatsapp, error) {
	var communicationWhatsapp models.CommunicationWhatsapp
	result := repo.DB.WithContext(ctx).
		Preload("Attachments").
		Preload("Instances").
		Preload("Instances.WhatsappInstance").
		Where("id = ?", id).
		First(&communicationWhatsapp)
	if result.Error != nil {
		return nil, result.Error
	}
	return &communicationWhatsapp, nil
}
