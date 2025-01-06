package models

type Lead struct {
	ID             int    `json:"id" gorm:"column:id;type:int"`
	OrganizationID int    `json:"organizationId" gorm:"column:organization_id;type:int"`
	Email          string `json:"email" gorm:"column:email;type:varchar(255)"`
	Phone          string `json:"phone" gorm:"column:phone;type:varchar(255)"`
	LanguageCode   string `json:"languageCode" gorm:"column:language_code;type:varchar(255)"`
}

func (Lead) TableName() string {
	return "leads"
}
