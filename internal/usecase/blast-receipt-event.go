package usecase

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"afrus-whatsapp-evolution_api-notification/internal/application/dto"
	"afrus-whatsapp-evolution_api-notification/internal/application/repositories"
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"afrus-whatsapp-evolution_api-notification/internal/services"
	"afrus-whatsapp-evolution_api-notification/pkg/queue"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

type ReceiptBlastEventUseCase struct {
	Ctx                   context.Context
	Configs               *config.Config
	Queue                 *queue.RabbitMQ
	AfrusDB               *gorm.DB
	EventsDB              *gorm.DB
	whatsappSenderService *services.WhatsappSenderService
}

func NewReceiptBlastEventUseCase(ctx context.Context, configs *config.Config, queue *queue.RabbitMQ, afrusDB, eventsDB *gorm.DB, whatsappSenderService *services.WhatsappSenderService) *ReceiptBlastEventUseCase {
	return &ReceiptBlastEventUseCase{
		Ctx:                   ctx,
		Configs:               configs,
		Queue:                 queue,
		AfrusDB:               afrusDB,
		EventsDB:              eventsDB,
		whatsappSenderService: whatsappSenderService,
	}
}

func (rbu *ReceiptBlastEventUseCase) Execute(event string) error {
	var data dto.BlastEventProcess
	if err := json.Unmarshal([]byte(event), &data); err != nil {
		log.Printf("Failed to unmarshal event: %v", err)
		return err
	}

	leadRepo := repositories.NewLeadRepository(rbu.AfrusDB)
	lead, err := leadRepo.FindById(rbu.Ctx, data.LeadID)
	if err != nil {
		return err
	}

	communicationWhatsappRepo := repositories.NewCommunicationWhatsappRepository(rbu.AfrusDB)
	communicationWhatsapp, err := communicationWhatsappRepo.FindById(rbu.Ctx, data.CommunicationWhatsappId)
	if err != nil {
		return err
	}

	var resp *services.WhatsappResponse

	for _, instance := range communicationWhatsapp.Instances {
		// err := rbu.processRules(&instance.WhatsappInstance)
		// if err != nil {
		// 	log.Printf("Error processing rules: %v - Trying with the next instance", err)
		// 	continue
		// }

		resp, err = rbu.sendMessage(data, &instance.WhatsappInstance, lead, communicationWhatsapp)
		if err != nil {
			log.Printf("Error sending message: %v - Trying with the next instance", err)
			continue
		}

		// If message is sent successfully, break the loop
		log.Printf("[BLAST] - Message sent successfully with instance: %v to: %s", instance.WhatsappInstance.InstanceName, lead.Email)

		rbu.StoreEvent("sent", data, lead, resp)

		break
	}

	return nil
}

func (rbu *ReceiptBlastEventUseCase) sendMessage(data dto.BlastEventProcess, instance *models.WhatsappInstance, lead *models.Lead, communication *models.CommunicationWhatsapp) (*services.WhatsappResponse, error) {
	log.Printf("[BLAST] - Sending message to: %s %s \n", lead.Email, lead.Phone)

	resp, err := rbu.whatsappSenderService.SendWhatsappTextMessage(lead, instance, data.Content)
	if err != nil {
		rbu.StoreEvent("failed", data, lead, nil)
		return nil, err
	}

	for _, attachment := range communication.Attachments {
		if attachment.Type == uint(services.LINK) {
			_, err := rbu.whatsappSenderService.SendWhatsappTextMessage(lead, instance, attachment.Content)
			if err != nil {
				log.Printf("[BLAST] - Error sending link attachment: %v", err)
			}
		} else {
			whatsappAttachment := services.WhatsappAttachement{
				Type:     attachment.Type,
				Content:  attachment.Content,
				Filename: attachment.Filename,
				Size:     attachment.Size,
			}
			_, err := rbu.whatsappSenderService.SendWhatsappMediaMessage(lead, instance, whatsappAttachment, attachment.Filename)
			if err != nil {
				log.Printf("[BLAST] - Error sending attachment: %v", err)
			}
		}
	}

	return resp, nil
}

func (rbu *ReceiptBlastEventUseCase) StoreEvent(kind string, data dto.BlastEventProcess, lead *models.Lead, resp *services.WhatsappResponse) error {
	eventRepo := repositories.NewWhatsappEventRepository(rbu.EventsDB)

	var messageID = ""
	if resp != nil {
		messageID = resp.Key.ID
	}

	var eventMap models.JSONB
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("error marshalling event response: %v", err)
	}
	if err := json.Unmarshal(respBytes, &eventMap); err != nil {
		return fmt.Errorf("error unmarshalling event response: %v", err)
	}

	event := &models.WhatsappEvent{
		LeadID:         data.LeadID,
		OrganizationID: data.OrganizationID,
		PhoneNumber:    lead.Phone,
		ExternalID:     strconv.Itoa(data.CommunicationWhatsappId),
		ExternalTable:  "communication_whatsapps",
		MessageID:      messageID,
		EventType:      1,
		DateEvent:      time.Now().Format(time.RFC3339),
		Event:          eventMap,
	}

	if err := eventRepo.Save(rbu.Ctx, kind, event); err != nil {
		return fmt.Errorf("[EVENT] - error saving event: %v", err)
	}

	log.Printf("[EVENT] - Event of type: '%s' for communicationWhatsappId: '%d' saved successfully", kind, data.CommunicationWhatsappId)

	return nil
}

func (rbu *ReceiptBlastEventUseCase) processRules(instance *models.WhatsappInstance) error {
	if err := rbu.maxConsecutivesSent(instance); err != nil {
		return err
	}
	if err := rbu.maxSentRate(instance); err != nil {
		return err
	}
	if err := rbu.sleepTime(); err != nil {
		return err
	}
	return nil
}

func (rbu *ReceiptBlastEventUseCase) maxConsecutivesSent(instance *models.WhatsappInstance) error {
	const (
		baseMaxSends  = 2
		maxSendsLimit = 6
	)

	monthsSinceCreation := int(time.Since(instance.CreatedAt).Hours() / (24 * 30))

	// Calculate max allowed sends (base + 1 per month, capped at maxSendsLimit)
	maxAllowedSends := baseMaxSends + monthsSinceCreation
	if maxAllowedSends > maxSendsLimit {
		maxAllowedSends = maxSendsLimit
	}

	currentSends, ok := instance.Data["consecutive_sends"].(float64)
	if !ok {
		instance.Data["consecutive_sends"] = baseMaxSends
		return nil
	}

	if int(currentSends) >= maxAllowedSends {
		return fmt.Errorf("max consecutive sends limit reached: %d/%d", int(currentSends), maxAllowedSends)
	}

	instance.Data["consecutive_sends"] = currentSends + 1
	return nil
}

func (rbu *ReceiptBlastEventUseCase) maxSentRate(instance *models.WhatsappInstance) error {
	const cooldownMinutes = 5

	lastSendStr, ok := instance.Data["last_send_time"].(string)
	if !ok {
		instance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
		return nil
	}

	lastSendTime, err := time.Parse(time.RFC3339, lastSendStr)
	if err != nil {
		return fmt.Errorf("invalid last send time format: %v", err)
	}

	// Check if enough time has passed
	if time.Since(lastSendTime).Minutes() < cooldownMinutes {
		return fmt.Errorf("message rate limit: the message was scheduled for %d", cooldownMinutes)
	}

	instance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
	return nil
}

func (rbu *ReceiptBlastEventUseCase) sleepTime() error {
	randomDelay := time.Duration(rand.Intn(60)+1) * time.Second
	time.Sleep(randomDelay)
	return nil
}
