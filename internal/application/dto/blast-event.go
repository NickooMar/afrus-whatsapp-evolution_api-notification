package dto

type BlastEventProcess struct {
	Content                 string `json:"content"`
	LeadID                  int    `json:"leadId"`
	CommunicationWhatsappId int    `json:"communicationWhatsappId"`
	OrganizationID          int    `json:"organizationId"`
}
