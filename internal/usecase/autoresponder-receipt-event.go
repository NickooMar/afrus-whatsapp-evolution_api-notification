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

type ReceiptAutoresponderEventUseCase struct {
	Ctx                   context.Context
	Configs               *config.Config
	Queue                 *queue.RabbitMQ
	AfrusDB               *gorm.DB
	EventsDB              *gorm.DB
	whatsappSenderService *services.WhatsappSenderService
}

func NewReceiptAutoresponderEventUseCase(ctx context.Context, configs *config.Config, queue *queue.RabbitMQ, afrusDB, eventsDB *gorm.DB, whatsappSenderService *services.WhatsappSenderService) *ReceiptAutoresponderEventUseCase {
	return &ReceiptAutoresponderEventUseCase{
		Ctx:                   ctx,
		Configs:               configs,
		Queue:                 queue,
		AfrusDB:               afrusDB,
		EventsDB:              eventsDB,
		whatsappSenderService: whatsappSenderService,
	}
}

func (rwe *ReceiptAutoresponderEventUseCase) Execute(event string) error {
	var data dto.AutoresponderEventProcess
	if err := json.Unmarshal([]byte(event), &data); err != nil {
		log.Printf("Error to decode JSON: %v", err)
		return err
	}

	leadRepo := repositories.NewLeadRepository(rwe.AfrusDB)
	lead, err := leadRepo.FindById(rwe.Ctx, data.LeadID)
	if err != nil {
		return err
	}

	whatsappInstanceRepo := repositories.NewWhatsappInstanceRepository(rwe.AfrusDB)
	whatsappInstance, err := whatsappInstanceRepo.GetWhatsappInstanceById(rwe.Ctx, data.WhatsappInstanceID)
	if err != nil {
		return err
	}

	whatsappInstances, err := whatsappInstanceRepo.GetWhatsappInstancesByOrganization(rwe.Ctx, whatsappInstance)
	if err != nil {
		return err
	}

	whatsappTriggerRepo := repositories.NewWhatsappTriggerRepository(rwe.AfrusDB)
	whatsappTrigger, err := whatsappTriggerRepo.GetWhatsappTriggerById(rwe.Ctx, data.WhatsappTriggerID)
	if err != nil {
		return err
	}

	whatsappTriggerAttachmentsRepo := repositories.NewWhatsappTriggerAttachmentRepository(rwe.AfrusDB)
	attachments, err := whatsappTriggerAttachmentsRepo.GetByTriggerId(rwe.Ctx, whatsappTrigger.ID)
	if err != nil {
		return err
	}

	if err := rwe.processRules(data, whatsappInstance, whatsappTrigger); err != nil {
		return err
	}

	if err := whatsappInstanceRepo.Update(rwe.Ctx, whatsappInstance); err != nil {
		return fmt.Errorf("error updating whatsapp instance data: %v", err)
	}

	var resp *services.WhatsappResponse

	// TODO: Move this to a helper function
	if len(attachments) == 0 {
		resp, err = rwe.whatsappSenderService.SendWhatsappTextMessage(lead, whatsappInstance, data.Content)
		if err != nil {
			fmt.Printf("Failed to send message in main instance %s - %v\n", whatsappInstance.InstanceName, err)
			for _, instance := range whatsappInstances {
				resp, err = rwe.whatsappSenderService.SendWhatsappTextMessage(lead, &instance, data.Content)
				if err != nil {
					fmt.Printf("[Failed to send message in %s] - %v\n", instance.InstanceName, err)
				} else {
					if err := rwe.StoreEvent("sent", data, lead, resp); err != nil {
						return err
					}
					break
				}
			}
			if err != nil {
				if storeErr := rwe.StoreEvent("failed", data, lead, resp); storeErr != nil {
					return storeErr
				}
				return err
			}
		} else {
			if err := rwe.StoreEvent("sent", data, lead, resp); err != nil {
				return err
			}
		}
	} else {
		for _, attachment := range attachments {
			whatsappAttachment := services.WhatsappAttachement{
				Type:     attachment.Type,
				Content:  attachment.Content,
				Filename: attachment.Filename,
				Size:     attachment.Size,
			}
			resp, err = rwe.whatsappSenderService.SendWhatsappMediaMessage(lead, whatsappInstance, whatsappAttachment, data.Content)
			if err != nil {
				fmt.Printf("[Failed to send media message to main instance %s] - %v\n", whatsappInstance.InstanceName, err)
				if storeErr := rwe.StoreEvent("failed", data, lead, resp); storeErr != nil {
					return storeErr
				}
				return err
			}
		}
		if err := rwe.StoreEvent("sent", data, lead, resp); err != nil {
			return err
		}
	}

	err = rwe.SendEventToBilling()
	if err != nil {
		return err
	}

	log.Printf("[MESSAGE] - Message processed successfully - [Name: %s / Owner: %s / To: %s / Lead: %s] \n", whatsappTrigger.Name, whatsappInstance.Owner, lead.Phone, lead.Email)

	return nil
}

func (rwe *ReceiptAutoresponderEventUseCase) processRules(data dto.AutoresponderEventProcess, whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	if err := rwe.maxConsecutivesSent(data, whatsappInstance, whatsappTrigger); err != nil {
		return err
	}
	if err := rwe.maxSentRate(data, whatsappInstance, whatsappTrigger); err != nil {
		return err
	}
	if err := rwe.sleepTime(); err != nil {
		return err
	}
	return nil
}

func (rwe *ReceiptAutoresponderEventUseCase) maxConsecutivesSent(data dto.AutoresponderEventProcess, whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	const (
		baseMaxSends  = 2
		maxSendsLimit = 6
	)

	monthsSinceCreation := int(time.Since(whatsappInstance.CreatedAt).Hours() / (24 * 30))

	// Calculate max allowed sends (base + 1 per month, capped at maxSendsLimit)
	maxAllowedSends := baseMaxSends + monthsSinceCreation
	if maxAllowedSends > maxSendsLimit {
		maxAllowedSends = maxSendsLimit
	}

	currentSends, ok := whatsappInstance.Data["consecutive_sends"].(float64)
	if !ok {
		whatsappInstance.Data["consecutive_sends"] = baseMaxSends
		return nil
	}

	if int(currentSends) >= maxAllowedSends {
		if rwe.Configs.Environment == "development" {
			return nil
		} else {
			message := &dto.AutoresponderEventProcess{
				Content:            data.Content,
				LeadID:             data.LeadID,
				OrganizationID:     data.OrganizationID,
				WhatsappInstanceID: int(whatsappInstance.ID),
				WhatsappTriggerID:  int(whatsappTrigger.ID),
			}

			messageBytes, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("error marshalling message: %v", err)
			}

			rwe.Queue.Schedule(
				rwe.Configs.EvolutionAPINotificationExchange,
				rwe.Configs.EvolutionAPINotificationAutoresponderRoutingKey,
				messageBytes,
				int(time.Minute)*5,
			)
		}
		return fmt.Errorf("max consecutive sends limit reached: %d/%d", int(currentSends), maxAllowedSends)
	}

	whatsappInstance.Data["consecutive_sends"] = currentSends + 1
	return nil
}

func (rwe *ReceiptAutoresponderEventUseCase) maxSentRate(data dto.AutoresponderEventProcess, whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	const cooldownMinutes = 5

	lastSendStr, ok := whatsappInstance.Data["last_send_time"].(string)
	if !ok {
		whatsappInstance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
		return nil
	}

	lastSendTime, err := time.Parse(time.RFC3339, lastSendStr)
	if err != nil {
		return fmt.Errorf("invalid last send time format: %v", err)
	}

	// Check if enough time has passed
	if time.Since(lastSendTime).Minutes() < cooldownMinutes {
		if rwe.Configs.Environment == "development" {
			return nil
		} else {
			message := &dto.AutoresponderEventProcess{
				Content:            data.Content,
				LeadID:             data.LeadID,
				OrganizationID:     data.OrganizationID,
				WhatsappInstanceID: int(whatsappInstance.ID),
				WhatsappTriggerID:  int(whatsappTrigger.ID),
			}

			messageBytes, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("error marshalling message: %v", err)
			}

			rwe.Queue.Schedule(
				rwe.Configs.EvolutionAPINotificationExchange,
				rwe.Configs.EvolutionAPINotificationAutoresponderRoutingKey,
				messageBytes,
				cooldownMinutes,
			)
		}

		return fmt.Errorf("message rate limit: the message was scheduled for %d", cooldownMinutes)
	}

	whatsappInstance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
	return nil
}

func (rwe *ReceiptAutoresponderEventUseCase) sleepTime() error {
	randomDelay := time.Duration(rand.Intn(60)+1) * time.Second
	time.Sleep(randomDelay)
	return nil
}

func (rwe *ReceiptAutoresponderEventUseCase) StoreEvent(kind string, data dto.AutoresponderEventProcess, lead *models.Lead, resp *services.WhatsappResponse) error {
	eventRepo := repositories.NewWhatsappEventRepository(rwe.EventsDB)

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
		ExternalID:     strconv.Itoa(data.WhatsappInstanceID),
		ExternalTable:  "whatsapp_triggers",
		MessageID:      messageID,
		EventType:      1,
		DateEvent:      time.Now().Format(time.RFC3339),
		Event:          eventMap,
	}

	if err := eventRepo.Save(rwe.Ctx, kind, event); err != nil {
		return fmt.Errorf("[EVENT] - error saving event: %v", err)
	}
	return nil
}

func (rwe *ReceiptAutoresponderEventUseCase) SendEventToBilling() error {
	err := rwe.Queue.Publish(rwe.Ctx, rwe.Configs.RabbitMQBillingExchange, rwe.Configs.RabbitMQBillingRoutingKey, []byte("receipt"))
	if err != nil {
		return err
	}
	return nil
}
