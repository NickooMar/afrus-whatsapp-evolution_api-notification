package usecase

import (
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	contentTypeJSON  = "application/json"
	mediaTypeUnknown = "unknown"
	mimeTypeDefault  = "application/octet-stream"
)

var mediaTypeMap = map[string]string{
	"image":    "image/png",
	"audio":    "audio/mpeg",
	"video":    "video/mp4",
	"document": "application/pdf",
}
var mediaTypeKeys = []string{"image", "audio", "video", "document"}

func (rwe *ReceiptWhatsappEventUseCase) sendRequest(requestUrl string, payloadBytes []byte) error {
	payload := strings.NewReader(string(payloadBytes))

	req, err := http.NewRequest("POST", requestUrl, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", contentTypeJSON)
	req.Header.Add("apikey", rwe.Configs.EvolutionAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (rwe *ReceiptWhatsappEventUseCase) SendWhatsappMediaMessage(lead *models.Lead, whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger, attachment models.WhatsappTriggerAttachment) error {
	requestUrl := fmt.Sprintf("%s/message/sendMedia/%s", strings.TrimSuffix(rwe.Configs.EvolutionAPIBaseURL, "/"), whatsappInstance.InstanceName)

	mediaType := mediaTypeUnknown
	mimeType := mimeTypeDefault

	if int(attachment.Type) < len(mediaTypeKeys) {
		mediaKey := mediaTypeKeys[int(attachment.Type)-1]
		mediaType = mediaKey
		mimeType = mediaTypeMap[mediaKey]
	}

	to := rwe.FormatLeadPhone(lead)

	body := Payload{
		Number: to,
		MediaMessage: &MediaMessage{
			MediaType: mediaType,
			MimeType:  mimeType,
			Caption:   whatsappTrigger.Content,
			Media:     attachment.Content,
			FileName:  attachment.Filename,
		},
		Options: Options{
			Delay:    0,
			Presence: "composing",
		},
	}

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return rwe.sendRequest(requestUrl, payloadBytes)
}

func (rwe *ReceiptWhatsappEventUseCase) SendWhatsappTextMessage(lead *models.Lead, whatsappInstance *models.WhatsappInstance, whatsappTrigger *models.WhatsappTrigger) error {
	requestUrl := fmt.Sprintf("%s/message/sendText/%s", strings.TrimSuffix(rwe.Configs.EvolutionAPIBaseURL, "/"), whatsappInstance.InstanceName)

	to := rwe.FormatLeadPhone(lead)

	body := Payload{
		Number: to,
		TextMessage: &TextMessage{
			Text: whatsappTrigger.Content,
		},
		Options: Options{
			Delay:       0,
			Presence:    "composing",
			LinkPreview: true,
		},
	}

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return rwe.sendRequest(requestUrl, payloadBytes)
}

func (rwe *ReceiptWhatsappEventUseCase) FormatLeadPhone(lead *models.Lead) string {
	lead.Phone = strings.ReplaceAll(lead.Phone, "+", "")
	lead.Phone = strings.ReplaceAll(lead.Phone, "-", "")
	lead.Phone = strings.ReplaceAll(lead.Phone, " ", "")
	return lead.Phone
}
