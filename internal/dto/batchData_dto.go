package dto

type BatchDataDTO struct {
	EAN         string `json:"ean"`
	Item        string `json:"item"`
	Description string `json:"description"`
	AisleFacing string `json:"aisle_facing"`
}
