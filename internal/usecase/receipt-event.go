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
	"strconv"
	"time"

	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

type MediaMessage struct {
	MediaType string `json:"mediatype"`
	MimeType  string `json:"mimetype"`
	Caption   string `json:"caption"`
	Media     string `json:"media"`
	FileName  string `json:"fileName"`
}

type TextMessage struct {
	Text string `json:"text"`
}

type Options struct {
	Delay       int    `json:"delay"`
	Presence    string `json:"presence"`
	LinkPreview bool   `json:"linkPreview"`
}

type Payload struct {
	Number       string        `json:"number"`
	MediaMessage *MediaMessage `json:"mediaMessage"`
	TextMessage  *TextMessage  `json:"textMessage"`
	Options      Options       `json:"options"`
}

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

	if err := rwe.processRules(whatsappInstance, whatsappTrigger); err != nil {
		return err
	}

	var resp *WhatsappResponse

	if len(attachments) == 0 {
		resp, err = rwe.SendWhatsappTextMessage(lead, whatsappInstance, whatsappTrigger)
		if err != nil {
			rwe.StoreEvent("failed", data, lead, resp)
			return err
		}

	} else {
		for _, attachment := range attachments {
			resp, err = rwe.SendWhatsappMediaMessage(lead, whatsappInstance, whatsappTrigger, attachment)
			if err != nil {
				rwe.StoreEvent("failed", data, lead, resp)
				return err
			}
		}
	}

	if err := whatsappInstanceRepo.Update(rwe.Ctx, whatsappInstance); err != nil {
		return fmt.Errorf("error updating whatsapp instance data: %v", err)
	}

	if err := rwe.StoreEvent("sent", data, lead, resp); err != nil {
		return err
	}

	log.Printf("[MESSAGE] - Message processed for trigger - [Name: %s / Phone: %s ] \n", whatsappTrigger.Name, whatsappInstance.Owner)

	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) processRules(whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	if err := rwe.maxConsecutivesSent(whatsappInstance, whatsappTrigger); err != nil {
		return err
	}
	if err := rwe.maxSentRate(whatsappInstance, whatsappTrigger); err != nil {
		return err
	}
	if err := rwe.sleepTime(); err != nil {
		return err
	}
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) maxConsecutivesSent(whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
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
			rwe.Queue.Schedule(rwe.Configs.EvolutionAPINotificationExchange, rwe.Configs.EvolutionAPINotificationRoutingKey, []byte(fmt.Sprintf(`{
				"whatsapp_trigger_id": %d,
				"whatsapp_instance_id": %d
				}`, whatsappTrigger.ID, whatsappInstance.ID)), int(time.Minute)*5)
		}
		return fmt.Errorf("max consecutive sends limit reached: %d/%d", int(currentSends), maxAllowedSends)
	}

	whatsappInstance.Data["consecutive_sends"] = currentSends + 1
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) maxSentRate(whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
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
			rwe.Queue.Schedule(rwe.Configs.EvolutionAPINotificationExchange, rwe.Configs.EvolutionAPINotificationRoutingKey, []byte(fmt.Sprintf(`{
				"whatsapp_trigger_id": %d,
				"whatsapp_instance_id": %d
			}`, whatsappTrigger.ID, whatsappInstance.ID)), cooldownMinutes)
		}

		return fmt.Errorf("message rate limit: the message was scheduled for %d", cooldownMinutes)
	}

	whatsappInstance.Data["last_send_time"] = time.Now().Format(time.RFC3339)
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) sleepTime() error {
	randomDelay := time.Duration(rand.Intn(60)+1) * time.Second
	time.Sleep(randomDelay)
	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) StoreEvent(kind string, data dto.EventProcess, lead *models.Lead, resp *WhatsappResponse) error {
	eventRepo := repositories.NewWhatsappEventRepository(rwe.EventsDB)

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
		MessageID:      resp.Key.ID,
		EventType:      1,
		DateEvent:      time.Now().Format(time.RFC3339),
		Event:          eventMap,
	}

	if err := eventRepo.Save(rwe.Ctx, kind, event); err != nil {
		return fmt.Errorf("error saving event: %v", err)
	}
	return nil
}
