package api

type LabelDTO struct {
	LabelID     string  `json:"label_id"`
	Item        string  `json:"item"`
	EAN         string  `json:"ean"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	AisleFacing string  `json:"aisle_facing"`
	Bay         string  `json:"bay"`
}

type LabelResponse []LabelDTO
