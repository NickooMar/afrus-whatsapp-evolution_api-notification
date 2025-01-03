package usecase

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"afrus-whatsapp-evolution_api-notification/internal/application/dto"
	"afrus-whatsapp-evolution_api-notification/internal/application/protocols"
	"afrus-whatsapp-evolution_api-notification/internal/application/repositories"
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

type ReceiptWhatsappEventUseCase struct {
	Ctx      context.Context
	Configs  *config.Config
	Queue    protocols.Queue
	AfrusDB  *gorm.DB
	EventsDB *gorm.DB
}

func NewReceiptWhatsappEventUseCase(ctx context.Context, configs *config.Config, queue protocols.Queue, afrusDB, eventsDB *gorm.DB) *ReceiptWhatsappEventUseCase {
	return &ReceiptWhatsappEventUseCase{
		Ctx:      ctx,
		Configs:  configs,
		Queue:    queue,
		AfrusDB:  afrusDB,
		EventsDB: eventsDB,
	}
}

func (rwe *ReceiptWhatsappEventUseCase) Execute(event string) error {
	var data dto.EventProcess
	if err := json.Unmarshal([]byte(event), &data); err != nil {
		log.Printf("Error to decode JSON: %v", err)
	}

	whatsappInstanceRepo := repositories.NewWhatsappInstanceRepository(rwe.AfrusDB)
	whatsappInstance, err := whatsappInstanceRepo.GetWhatsappInstanceById(rwe.Ctx, data.WhatsappInstanceID)
	if err != nil {
		return err
	}

	whatsappTriggerRepo := repositories.NewWhatsappTriggerRepository(rwe.AfrusDB)
	whatsappTrigger, err := whatsappTriggerRepo.GetWhatsappTriggerById(rwe.Ctx, data.WhatsappTriggerID)
	if err != nil {
		return err
	}

	if err := rwe.processRules(whatsappInstance, whatsappTrigger); err != nil {
		return err
	}

	if err := rwe.SendWhatsappMessage(whatsappInstance, whatsappTrigger); err != nil {
		return err
	}

	if err := whatsappInstanceRepo.Update(rwe.Ctx, whatsappInstance); err != nil {
		return fmt.Errorf("error updating whatsapp instance data: %v", err)
	}

	log.Printf("[MESSAGE] - Message processed for trigger - [Name: %s / Phone: %s ] \n", whatsappTrigger.Name, whatsappInstance.Owner)

	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) processRules(whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	if err := rwe.maxConsecutivesSent(whatsappInstance); err != nil {
		return err
	}
	if err := rwe.maxSentRate(whatsappInstance); err != nil {
		return err
	}
	// if err := rwe.sleepTime(); err != nil {
	// 	return err
	// }

	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) maxConsecutivesSent(whatsappInstance *models.WhatsappInstance) error {
	const (
		baseMaxSends  = 2
		maxSendsLimit = 6
	)

	// Calculate months since creation
	monthsSinceCreation := int(time.Since(whatsappInstance.CreatedAt).Hours() / (24 * 30))

	// Calculate max allowed sends (base + 1 per month, capped at maxSendsLimit)
	maxAllowedSends := baseMaxSends + monthsSinceCreation
	if maxAllowedSends > maxSendsLimit {
		maxAllowedSends = maxSendsLimit
	}

	// Get current consecutive sends from instance data
	currentSends, ok := whatsappInstance.Data["consecutive_sends"].(float64)
	if !ok {
		// Initialize if not exists
		whatsappInstance.Data["consecutive_sends"] = baseMaxSends
		return nil
	}

	if int(currentSends) >= maxAllowedSends {
		return fmt.Errorf("max consecutive sends limit reached: %d/%d", int(currentSends), maxAllowedSends)
	}

	// Increment consecutive sends counter
	whatsappInstance.Data["consecutive_sends"] = currentSends + 1
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) maxSentRate(whatsappInstance *models.WhatsappInstance) error {
	const cooldownMinutes = 5

	// Get last send time from instance data
	lastSendStr, ok := whatsappInstance.Data["last_send_time"].(string)
	if !ok {
		// First message, set current time and allow
		whatsappInstance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
		return nil
	}

	lastSendTime, err := time.Parse(time.RFC3339, lastSendStr)
	if err != nil {
		return fmt.Errorf("invalid last send time format: %v", err)
	}

	// Check if enough time has passed
	if time.Since(lastSendTime).Minutes() < cooldownMinutes {
		return fmt.Errorf("message rate limit: please wait %d minutes between messages", cooldownMinutes)
	}

	// Update last send time
	whatsappInstance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) sleepTime() error {
	randomDelay := time.Duration(rand.Intn(60)+1) * time.Second
	time.Sleep(randomDelay)
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) SendWhatsappMessage(whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	// // Send message to Whatsapp API
	// if err := rwe.Queue.Publish(rwe.Configs.RabbitMQ.WhatsappQueue, whatsappInstance.Owner); err != nil {
	// 	return fmt.Errorf("error sending message to Whatsapp API: %v", err)
	// }

	return nil
}
