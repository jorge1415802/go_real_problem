package domain

import (
	"context"
	"strings"
)

type SalaLocation struct {
	EAN         string `json:"ean"`
	SKU         string `json:"sku"`
	Aisle       string `json:"aisle"`
	Section     string `json:"section"`
	Tray        string `json:"tray"`
	Description string `json:"description"`
	HasPosition bool   `json:"has_position"`
}

type StoreInfo struct {
	StoreId          string `json:"store_id"`
	CommercialPolicy int    `json:"commercial_policy"`
}

type SalaLocationRepository interface {
	DeleteSalaPositions(ctx context.Context, storeID string, policy int) error
	GetOrCreateProduct(ctx context.Context, sku, ean string) (int, error)
	GetOrCreateAisle(ctx context.Context, storeID, aisle string, policy int) (string, error)
	BulkInsertQrDarkstore(ctx context.Context, qrID string, productIDs []int) error
	UpdateQrUpdatedAt(ctx context.Context, qrIds []string) error
	GetStoresIds(ctx context.Context) ([]StoreInfo, error)
}

func NewSalaLocation(ean, sku, description, aisleFacing string) *SalaLocation {

	cleanEAN := strings.TrimLeft(ean, "0")
	cleanSKU := strings.TrimLeft(sku, "0")

	if len(cleanEAN) >= 7 {
		prefix := cleanEAN[:2]
		if prefix == "24" || prefix == "29" {
			cleanEAN = cleanEAN[:7]
		}
	}

	loc := &SalaLocation{
		EAN:         cleanEAN,
		SKU:         cleanSKU,
		Description: description,
		HasPosition: false,
	}

	if aisleFacing != "" {
		parts := strings.Split(aisleFacing, "-")
		loc.Aisle = getPart(parts, 0)
		loc.Section = getPart(parts, 1)
		loc.Tray = getPart(parts, 2)
		loc.HasPosition = true
	}

	return loc
}

func getPart(parts []string, index int) string {
	if index < len(parts) {
		return parts[index]
	}
	return ""
}
