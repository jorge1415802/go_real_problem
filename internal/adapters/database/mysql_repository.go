package database

import (
	"context"
	"cron/internal/domain"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
)

type ProductCacheKey struct {
	SKU string
	EAN string
}

type AisleCacheKey struct {
	STOREID    string
	POSITIONQR string
	POLICY     int
}

type MySQLRepository struct {
	db           *sql.DB
	aisleCache   map[AisleCacheKey]string
	productCache map[ProductCacheKey]int
	cacheMux     sync.RWMutex
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{
		db:           db,
		aisleCache:   make(map[AisleCacheKey]string),
		productCache: make(map[ProductCacheKey]int),
	}
}

func (r *MySQLRepository) GetOrCreateAisle(ctx context.Context, storeId, positionQr string, policy int) (string, error) {
	// logger := observability.GetLogger()
	r.cacheMux.RLock()
	id, exists := r.aisleCache[AisleCacheKey{STOREID: storeId, POSITIONQR: positionQr, POLICY: policy}]
	r.cacheMux.RUnlock()
	if exists {
		return id, nil
	}

	r.cacheMux.Lock()
	defer r.cacheMux.Unlock()

	if id, exists := r.aisleCache[AisleCacheKey{STOREID: storeId, POSITIONQR: positionQr, POLICY: policy}]; exists {
		return id, nil
	}

	var dbId string
	querySelect := `SELECT id FROM qr WHERE position_qr = ? AND sector_picking_id = 154 AND comercial_policy = ? LIMIT 1`
	err := r.db.QueryRowContext(ctx, querySelect, positionQr, policy).Scan(&dbId)

	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error getting id from QR on function GetOrCreateAisle %s", err)
		return "", fmt.Errorf("Error getting id from QR on function GetOrCreateAisle: %w", err)
	}

	if err == sql.ErrNoRows {
		location, err := time.LoadLocation("America/Santiago")
		if err != nil {
			fmt.Printf("Error loading location America/Santiago on function GetOrCreateAisle %s", err)
			return "", fmt.Errorf("Error loading location America/Santiago on function GetOrCreateAisle: %w", err)
		}
		now := time.Now().In(location)
		base := fmt.Sprintf("%s%s", storeId, now.Format("20060102"))
		nanoPart := now.UnixNano() % 1000000
		newId := fmt.Sprintf("%s%06d", base, nanoPart)

		queryInsert := `INSERT INTO qr (id, store_id, aisle, sector_picking_id, position_qr, status, comercial_policy, createdat, updatedat,user_id,auditat,audit_user_id) 
                        VALUES (?, ?, NULL, 154, ?, 3, ?, ?, ?, 'cron_zippedi', ?, 'cron_zippedi')`

		_, err = r.db.ExecContext(ctx, queryInsert,
			newId,      // id (el identificador largo)
			storeId,    // store_id
			positionQr, // position_qr
			policy,     // comercial_policy
			now,        // createdat
			now,        // updatedat
			now,        // auditat
		)
		if err != nil {
			fmt.Printf("Error inserting new QR on function GetOrCreateAisle %s", err)
			return "", fmt.Errorf("Error inserting new QR on function GetOrCreateAisle: %w", err)
		}

		dbId = newId
	}

	r.aisleCache[AisleCacheKey{STOREID: storeId, POSITIONQR: positionQr, POLICY: policy}] = dbId
	return dbId, nil
}

func (r *MySQLRepository) GetOrCreateProduct(ctx context.Context, sku, ean string) (int, error) {
	r.cacheMux.RLock()
	id, exists := r.productCache[ProductCacheKey{SKU: sku, EAN: ean}]
	r.cacheMux.RUnlock()
	if exists {
		return id, nil
	}
	var productId int
	querySelect := `SELECT id FROM products_darkstore WHERE sku = ? AND ean = ? LIMIT 1`
	err := r.db.QueryRowContext(ctx, querySelect, sku, ean).Scan(&productId)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("Error querying product on function GetOrCreateProduct: %w", err)
	}
	if err == sql.ErrNoRows {
		location, err := time.LoadLocation("America/Santiago")
		if err != nil {
			return 0, fmt.Errorf("Error loading location America/Santiago on function GetOrCreateProduct: %w", err)
		}
		now := time.Now().In(location)
		queryInsert := `INSERT INTO products_darkstore (sku, ean, createdat, updatedat) VALUES (?, ?, ?, ?)`
		result, err := r.db.ExecContext(ctx, queryInsert, sku, ean, now, now)
		if err != nil {
			return 0, fmt.Errorf("Error inserting product on function GetOrCreateProduct: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("Error getting last insert id on function GetOrCreateProduct: %w", err)
		}
		productId = int(id)
	}
	r.cacheMux.Lock()
	r.productCache[ProductCacheKey{SKU: sku, EAN: ean}] = productId
	r.cacheMux.Unlock()
	return productId, nil
}

func (r *MySQLRepository) BulkInsertQrDarkstore(ctx context.Context, qrId string, productsId []int) error {
	if len(productsId) == 0 || qrId == "" {
		return nil
	}

	location, err := time.LoadLocation("America/Santiago")
	if err != nil {
		return fmt.Errorf("Error loading location America/Santiago on function BulkInsertQrDarkstore: %w", err)
	}
	now := time.Now().In(location)
	query := `INSERT INTO qr_darkstore (numqr_id, producto_darkstore_id, createdat, updatedat, faces_quantity) VALUES`
	values := make([]interface{}, 0, len(productsId)*2)

	for _, productId := range productsId {
		query += ` (?, ?, ?, ?, 0),`
		values = append(values, qrId, productId, now, now)
	}

	query = query[:len(query)-1] // Quitar coma

	query += ` ON DUPLICATE KEY UPDATE 
				updatedat = ?,
				faces_quantity = 0`
	values = append(values, now)

	res, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		fmt.Printf("Error bulk inserting qr_darkstore on function BulkInsertQrDarkstore %s", err)
		return fmt.Errorf("Error bulk inserting qr_darkstore on function BulkInsertQrDarkstore: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	fmt.Printf("Inserted/Updated rows in qr_darkstore for QR ID %d %s", rowsAffected, qrId)
	return nil
}

func (r *MySQLRepository) UpdateQrUpdatedAt(ctx context.Context, qrIds []string) error {
	if len(qrIds) == 0 {
		return nil
	}

	location, err := time.LoadLocation("America/Santiago")
	if err != nil {
		return fmt.Errorf("Error loading location America/Santiago on function UpdateQrUpdatedAt: %w", err)
	}
	now := time.Now().In(location)

	const batchSize = 2000
	totalIds := len(qrIds)

	for i := 0; i < totalIds; i += batchSize {
		end := i + batchSize
		if end > totalIds {
			end = totalIds
		}
		// get the current batch of QR IDs to update
		batch := qrIds[i:end]

		// 1. Build the base query for this batch
		query := `UPDATE qr SET updatedat = ? WHERE id IN (`

		// 2. Initialize the slice with the exact size: 1 (for now) + number of elements in the batch
		values := make([]interface{}, 0, 1+len(batch))
		values = append(values, now) // Primer placeholder siempre es el 'updatedat'

		// 3. Inject the placeholders and the values in perfect sync
		for _, qrId := range batch {
			query += "?,"
			values = append(values, qrId)
		}
		query = query[:len(query)-1] // Remove the trailing comma
		query += ")"

		// 4. Execute safely in the database
		res, err := r.db.ExecContext(ctx, query, values...)
		if err != nil {
			fmt.Printf("Error updating batch in qr table %s", err)
			return fmt.Errorf("Error bulk updating qr table at batch [%d:%d]: %w", i, end, err)
		}

		rowsAffected, _ := res.RowsAffected()
		fmt.Printf("Batch updated successfully in qr table rowsAffected: %d batchSize: %d", rowsAffected, len(batch))
	}
	return nil
}

func (r *MySQLRepository) DeleteSalaPositions(ctx context.Context, storeID string, policy int) error {
	query := `DELETE FROM qr_darkstore 
				WHERE numqr_id IN (
					SELECT q.id 
					FROM qr q
					INNER JOIN darkstores d ON d.store_id = ? 
					WHERE q.sector_picking_id = 154 
					AND q.comercial_policy = ?
					AND q.store_id = ?
					AND d.flag_zippedi = 1
				)`

	res, err := r.db.ExecContext(ctx, query, storeID, policy, storeID)
	if err != nil {
		fmt.Printf("Error deleting qr_darkstore on function DeleteSalaPositions error: %s storeid: %s policy: %d", err, storeID, policy)
		return fmt.Errorf("Error deleting qr_darkstore on function DeleteSalaPositions: %w", err)
	}

	count, _ := res.RowsAffected()
	fmt.Printf("Deleted old sala positions: %d storeID: %s policy: %d", count, storeID, policy)
	return nil
}

func (r *MySQLRepository) GetStoresIds(ctx context.Context) ([]domain.StoreInfo, error) {
	query := `SELECT DISTINCT store_id, comercial_policy 
		FROM darkstores 
		WHERE flag_zippedi = 1`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("Error getting store IDs on function GetStoresIds: %w", err)
	}
	defer rows.Close()

	var storesInfos []domain.StoreInfo

	for rows.Next() {
		var storeID string
		var comercialPolicy int
		if err := rows.Scan(&storeID, &comercialPolicy); err != nil {
			return nil, fmt.Errorf("Error scanning store ID on function GetStoresIds: %w", err)
		}
		if strings.TrimSpace(storeID) != "" {
			storesInfos = append(storesInfos, domain.StoreInfo{StoreId: storeID, CommercialPolicy: comercialPolicy})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating over store IDs on function GetStoresIds: %w", err)
	}

	return storesInfos, nil
}
