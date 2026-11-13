package ports

import (
	"context"
	"cron/internal/dto"
)

type BatchProvider interface {
	Login(ctx context.Context, username, password string) error
	GetSession(ctx context.Context, storeId string) (string, error)
	GetLabels(ctx context.Context, storeId, session string) ([]dto.BatchDataDTO, error)
}
