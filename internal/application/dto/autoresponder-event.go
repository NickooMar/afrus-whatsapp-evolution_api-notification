package dto

type AutoresponderEventProcess struct {
	Content            string `json:"content"`
	LeadID             int    `json:"lead_id"`
	OrganizationID     int    `json:"organization_id"`
	WhatsappTriggerID  int    `json:"whatsapp_trigger_id"`
	WhatsappInstanceID int    `json:"whatsapp_instance_id"`
}
