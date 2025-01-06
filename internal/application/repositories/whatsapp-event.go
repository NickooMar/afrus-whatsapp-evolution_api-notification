package repositories

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type WhatsappEventRepository struct {
	DB *gorm.DB
}

type WhatsappEventRepositoryInterface interface {
	Save(dbName string, whatsappEvent models.WhatsappEvent) (int, error)
}

func NewWhatsappEventRepository(db *gorm.DB) *WhatsappEventRepository {
	return &WhatsappEventRepository{DB: db}
}

func (repo *WhatsappEventRepository) Save(ctx context.Context, dbName string, whatsappEvent models.WhatsappEvent) error {
	tableName := fmt.Sprintf("whatsapp.%s", dbName)
	result := repo.DB.Table(tableName).Create(&whatsappEvent)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
