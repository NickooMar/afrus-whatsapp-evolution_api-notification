package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"

	"gorm.io/gorm"
)

type LeadRepository struct {
	DB *gorm.DB
}

type LeadRepositoryInterface interface {
	FindById(ctx context.Context, id int) (*models.Lead, error)
}

func NewLeadRepository(db *gorm.DB) *LeadRepository {
	return &LeadRepository{DB: db}
}

func (repo *LeadRepository) FindById(ctx context.Context, id int) (*models.Lead, error) {
	var lead models.Lead
	result := repo.DB.WithContext(ctx).Where("id = ?", id).First(&lead)
	if result.Error != nil {
		return nil, result.Error
	}
	return &lead, nil
}
