package services

import (
	config "afrus-whatsapp-evolution_api-notification/configs"
	"afrus-whatsapp-evolution_api-notification/internal/domain/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Key struct {
	RemoteJid string `json:"remoteJid"`
	FromMe    bool   `json:"fromMe"`
	ID        string `json:"id"`
}

type ContextInfo struct{}

type ExtendedTextMessage struct {
	Text        string      `json:"text"`
	ContextInfo ContextInfo `json:"contextInfo"`
}

type Message struct {
	ExtendedTextMessage ExtendedTextMessage `json:"extendedTextMessage"`
}

type WhatsappResponse struct {
	Key              Key     `json:"key"`
	Message          Message `json:"message"`
	MessageTimestamp string  `json:"messageTimestamp"`
	Status           string  `json:"status"`
}

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

type WhatsappMediaType int

const (
	URL  WhatsappMediaType = 1
	FILE WhatsappMediaType = 2
	LINK WhatsappMediaType = 3
)

type WhatsappAttachement struct {
	Content  string
	Filename string
	Size     int
	Type     uint
}

type WhatsappSenderService struct {
	Configs *config.Config
}

func NewWhatsappSenderService(configs *config.Config) *WhatsappSenderService {
	return &WhatsappSenderService{Configs: configs}
}

func (wss *WhatsappSenderService) sendRequest(requestUrl string, payloadBytes []byte) (*WhatsappResponse, error) {
	payload := strings.NewReader(string(payloadBytes))

	req, err := http.NewRequest("POST", requestUrl, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("apikey", wss.Configs.EvolutionAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var whatsappResponse WhatsappResponse
	if err := json.Unmarshal(bodyBytes, &whatsappResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &whatsappResponse, nil
}

func (wss *WhatsappSenderService) SendWhatsappTextMessage(lead *models.Lead, instance *models.WhatsappInstance, content string) (*WhatsappResponse, error) {
	requestUrl := fmt.Sprintf("%s/message/sendText/%s", strings.TrimSuffix(wss.Configs.EvolutionAPIBaseURL, "/"), instance.InstanceName)

	to := wss.FormatLeadPhone(lead)

	body := Payload{
		Number: to,
		TextMessage: &TextMessage{
			Text: content,
		},
		Options: Options{
			Delay:       0,
			Presence:    "composing",
			LinkPreview: true,
		},
	}

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return wss.sendRequest(requestUrl, payloadBytes)
}

func (wss *WhatsappSenderService) SendWhatsappMediaMessage(lead *models.Lead, instance *models.WhatsappInstance, attachment WhatsappAttachement, content string) (*WhatsappResponse, error) {
	requestUrl := fmt.Sprintf("%s/message/sendMedia/%s", strings.TrimSuffix(wss.Configs.EvolutionAPIBaseURL, "/"), instance.InstanceName)

	mediaType := mediaTypeUnknown
	mimeType := mimeTypeDefault

	if int(attachment.Type) < len(mediaTypeKeys) {
		mediaKey := mediaTypeKeys[int(attachment.Type)-1]
		mediaType = mediaKey
		mimeType = mediaTypeMap[mediaKey]
	}

	to := wss.FormatLeadPhone(lead)

	body := Payload{
		Number: to,
		MediaMessage: &MediaMessage{
			MediaType: mediaType,
			MimeType:  mimeType,
			Caption:   content,
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
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return wss.sendRequest(requestUrl, payloadBytes)
}

func (wss *WhatsappSenderService) FormatLeadPhone(lead *models.Lead) string {
	lead.Phone = strings.ReplaceAll(lead.Phone, "+", "")
	lead.Phone = strings.ReplaceAll(lead.Phone, "-", "")
	lead.Phone = strings.ReplaceAll(lead.Phone, " ", "")
	return lead.Phone
}
