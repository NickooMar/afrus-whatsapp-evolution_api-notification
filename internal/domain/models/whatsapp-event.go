package models

type WhatsappEvent interface {
	GetID() int
	SetID(id int)
	TableName() string
}
