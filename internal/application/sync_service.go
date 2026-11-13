package application

import (
	"context"
	"cron/internal/domain"
	"cron/internal/ports"
	"fmt"
	"strings"
	"sync"
)

type SyncService struct {
	provider ports.BatchProvider
	repo     domain.SalaLocationRepository
}

func NewSyncService(z ports.BatchProvider, r domain.SalaLocationRepository) *SyncService {
	return &SyncService{
		provider: z,
		repo:     r,
	}
}

func (s *SyncService) ProcessStore(ctx context.Context, storeID string, policy int) {

	if err := s.repo.DeleteSalaPositions(ctx, storeID, policy); err != nil {
		fmt.Printf("Error cleaning sala for store: %s storeID: %s", err, storeID)
		return
	}

	session, err := s.provider.GetSession(ctx, storeID)
	if err != nil {
		fmt.Printf("Error getting session for store: %s storeID: %s", err, storeID)
		return
	}

	labels, err := s.provider.GetLabels(ctx, storeID, session)
	if err != nil {
		fmt.Printf("Error getting labels for store: %s storeID: %s", err, storeID)
		return
	}

	batches := make(map[string][]int)

	qrIds := make([]string, 0)

	for _, l := range labels {
		pID, _ := s.repo.GetOrCreateProduct(ctx, l.Item, l.EAN)

		parts := strings.Split(l.AisleFacing, "-")
		if len(parts) > 0 {
			aisle := parts[0]
			batches[aisle] = append(batches[aisle], pID)
		}
	}

	for aisle, pIDs := range batches {
		qrID, _ := s.repo.GetOrCreateAisle(ctx, storeID, aisle, policy)
		s.repo.BulkInsertQrDarkstore(ctx, qrID, pIDs)
		qrIds = append(qrIds, qrID)
	}

	s.repo.UpdateQrUpdatedAt(ctx, qrIds)
	fmt.Printf("Sync completed for store: %s storeID: %s policy: %d", storeID, storeID, policy)
}

func (s *SyncService) RunMultiStoreSync(ctx context.Context, storesInfos []domain.StoreInfo, user, password string) {
	if len(storesInfos) == 0 {
		fmt.Println("No storesInfos provided")
		return
	}

	if err := s.provider.Login(ctx, user, password); err != nil {
		fmt.Printf("Fails to login error: %s", err)
		return
	}

	jobs := make(chan domain.StoreInfo, len(storesInfos))
	var wg sync.WaitGroup
	numWorkers := 6

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for storeInfo := range jobs {
				fmt.Printf("[Worker] Processing store %s and comercial policy: %d\n", storeInfo.StoreId, storeInfo.CommercialPolicy)
				s.ProcessStore(ctx, storeInfo.StoreId, storeInfo.CommercialPolicy)
			}
		}()
	}

	for _, storeInfo := range storesInfos {
		jobs <- storeInfo
	}
	close(jobs)

	wg.Wait()
	fmt.Println("--- Sync MultiStore completed ---")
}
