package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type OrderItem struct {
	ProductID   int
	ProductName string
	Image       string
	Quantity    int
	Price       int
	IsReviewed  bool
}

type Order struct {
	OrderID        string
	Date           string
	CompletedAt    string // ✅ Ditambahkan
	BuyerName      string
	BuyerProfile   string
	Total          int
	Status         string
	ShippingCost   int
	PaymentMethod  string
	PaymentProof   string
	Email          string
	Phone          string
	Address        string
	Items          []OrderItem
	RefundBank     string
	RefundAccount  string
	RefundName     string
	TrackingDetail string
}

type CreateOrderInput struct {
	UserID         int
	BuyerName      string
	Address        string
	Phone          string
	Email          string
	PaymentMethod  string
	TotalPrice     int
	ShippingCost   int
	Status         string
	Items          []OrderItemInput
}

type OrderItemInput struct {
	ProductID int
	Quantity  int
	Price     int
}

type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func isJabodetabek(address string) bool {
	addr := strings.ToLower(address)
	keywords := []string{"jakarta", "bogor", "depok", "tangerang", "bekasi"}
	for _, k := range keywords {
		if strings.Contains(addr, k) {
			return true
		}
	}
	return false
}

// ================== SYSTEM TRACKING BARU (HISTORY BASED) ==================

func (r *OrderRepository) AppendTracking(orderID string, newStep string) error {
	var current string

	err := r.DB.QueryRow(
		"SELECT tracking_detail FROM orders WHERE order_id = ?",
		orderID,
	).Scan(&current)

	if err != nil {
		return err
	}

	if current == "" {
		current = newStep
	} else {
		current = current + " | " + newStep
	}

	_, err = r.DB.Exec(
		"UPDATE orders SET tracking_detail = ? WHERE order_id = ?",
		current, orderID,
	)

	return err
}

func (r *OrderRepository) UpdateTrackingStep(orderID string) error {
	var address string
	var current string

	err := r.DB.QueryRow(
		"SELECT address, tracking_detail FROM orders WHERE order_id = ?",
		orderID,
	).Scan(&address, &current)

	if err != nil {
		return err
	}

	jabodetabek := isJabodetabek(address)
	steps := []string{}

	// ✅ UPDATE: Menambahkan "Pesanan sedang dibuat" agar sinkron dengan database
	if jabodetabek {
		steps = []string{
			"Pesanan sedang dibuat",
			"Pesanan telah diserahkan ke jasa kirim",
			"Pesanan telah sampai di DC kota tujuan",
			"Pesanan sedang diantar ke alamat tujuan",
			"Pesanan telah sampai ke alamat tujuan",
		}
	} else {
		steps = []string{
			"Pesanan sedang dibuat",
			"Pesanan telah diserahkan ke jasa kirim",
			"Pesanan telah sampai di DC provinsi",
			"Pesanan telah sampai di DC kota",
			"Pesanan sedang diantar ke alamat tujuan",
			"Pesanan telah sampai ke alamat tujuan",
		}
	}

	var currentSteps []string
	if strings.TrimSpace(current) == "" {
		currentSteps = []string{}
	} else {
		currentSteps = strings.Split(current, " | ")
	}

	if len(currentSteps) >= len(steps) {
		return nil
	}

	lastStep := ""
	if len(currentSteps) > 0 {
		lastStep = currentSteps[len(currentSteps)-1]
	}

	if strings.Contains(strings.ToLower(lastStep), "sampai ke alamat") {
		return nil
	}

	nextStep := steps[len(currentSteps)]

	err = r.AppendTracking(orderID, nextStep)
	if err != nil {
		return err
	}

	if nextStep == "Pesanan telah sampai ke alamat tujuan" {
		_, _ = r.DB.Exec(`
            UPDATE orders 
            SET status = 'Selesai', completed_at = CURRENT_TIMESTAMP 
            WHERE order_id = ?
        `, orderID)
	}

	return nil
}

// ===========================================================================

func (r *OrderRepository) CreateOrder(input CreateOrderInput) (string, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return "", err
	}

	orderID := fmt.Sprintf("HM-%d", time.Now().UnixNano()/1e6)
	trackingInfo := "Pesanan sedang dibuat"

	queryOrder := `
        INSERT INTO orders (
            order_id, user_id, buyer_name, address, phone, email,
            total_price, shipping_cost, payment_method, status, created_at, tracking_detail
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)`

	_, err = tx.Exec(queryOrder,
		orderID,
		input.UserID,
		input.BuyerName,
		input.Address,
		input.Phone,
		input.Email,
		input.TotalPrice,
		input.ShippingCost,
		input.PaymentMethod,
		input.Status,
		trackingInfo,
	)

	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("gagal insert order: %v", err)
	}

	for _, item := range input.Items {
		queryItem := `INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES (?, ?, ?, ?)`
		_, err = tx.Exec(queryItem, orderID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("gagal insert order item: %v", err)
		}

		_, err = tx.Exec("UPDATE products SET stock = stock - ?, sold = sold + ? WHERE id = ?",
			item.Quantity, item.Quantity, item.ProductID)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("gagal update stok: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("gagal commit transaksi: %v", err)
	}

	return orderID, nil
}

func (r *OrderRepository) GetAllOrders() ([]Order, error) {
	query := `
        SELECT 
            o.order_id, o.created_at, COALESCE(o.completed_at, ''), o.total_price, o.shipping_cost, o.status, 
            o.buyer_name, o.address, o.payment_method, COALESCE(o.payment_proof, ''),
            COALESCE(o.email, ''), COALESCE(o.phone, ''),
            COALESCE(o.refund_bank, ''), COALESCE(o.refund_account, ''), COALESCE(o.refund_name, ''),
            COALESCE(u.profile_picture, ''), COALESCE(o.tracking_detail, '')
        FROM orders o
        LEFT JOIN users u ON o.user_id = u.id
        ORDER BY o.created_at DESC`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var rawTime, rawCompleted string
		err := rows.Scan(
			&o.OrderID, &rawTime, &rawCompleted, &o.Total, &o.ShippingCost, &o.Status,
			&o.BuyerName, &o.Address, &o.PaymentMethod, &o.PaymentProof,
			&o.Email, &o.Phone,
			&o.RefundBank, &o.RefundAccount, &o.RefundName,
			&o.BuyerProfile, &o.TrackingDetail,
		)
		if err == nil {
			o.Date = parseOrderDate(rawTime)
			if rawCompleted != "" {
				o.CompletedAt = parseOrderDate(rawCompleted)
			} else {
				o.CompletedAt = "-"
			}
			items, _ := r.getOrderItems(o.OrderID)
			o.Items = items
			orders = append(orders, o)
		}
	}
	return orders, nil
}

func (r *OrderRepository) GetOrderByID(orderID string) (Order, error) {
	var o Order
	var rawTime, rawCompleted string

	queryOrder := `
        SELECT 
            o.order_id, o.created_at, COALESCE(o.completed_at, ''), o.total_price, o.shipping_cost, o.status, 
            o.payment_method, COALESCE(o.payment_proof, ''), COALESCE(o.buyer_name, ''), 
            COALESCE(o.address, ''), COALESCE(o.email, ''),  COALESCE(o.phone, ''),
            COALESCE(o.refund_bank, ''), COALESCE(o.refund_account, ''), COALESCE(o.refund_name, ''),
            COALESCE(u.profile_picture, ''), COALESCE(o.tracking_detail, '')
        FROM orders o
        LEFT JOIN users u ON o.user_id = u.id
        WHERE o.order_id = ?`

	err := r.DB.QueryRow(queryOrder, orderID).Scan(
		&o.OrderID, &rawTime, &rawCompleted, &o.Total, &o.ShippingCost, &o.Status,
		&o.PaymentMethod, &o.PaymentProof, &o.BuyerName, &o.Address, &o.Email, &o.Phone,
		&o.RefundBank, &o.RefundAccount, &o.RefundName,
		&o.BuyerProfile, &o.TrackingDetail,
	)
	if err != nil {
		return o, err
	}
	o.Date = parseOrderDate(rawTime)
	if rawCompleted != "" {
		o.CompletedAt = parseOrderDate(rawCompleted)
	} else {
		o.CompletedAt = "-"
	}

	items, _ := r.getOrderItems(o.OrderID)
	o.Items = items

	return o, nil
}

func (r *OrderRepository) UpdateOrderStatus(orderID string, status string) error {
	_, err := r.DB.Exec(
		`UPDATE orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE order_id = ?`,
		status, orderID,
	)
	if err != nil {
		return err
	}

	// ✅ TAMBAHAN PENTING DI SINI
	if status == "Dikirim" {
		var current string

		err := r.DB.QueryRow(
			"SELECT tracking_detail FROM orders WHERE order_id = ?",
			orderID,
		).Scan(&current)

		if err == nil {
			if !strings.Contains(strings.ToLower(current), "diserahkan ke jasa kirim") {
				_ = r.AppendTracking(orderID, "Pesanan telah diserahkan ke jasa kirim")
			}
		}
	}

	if status == "Selesai" {
		_, _ = r.DB.Exec(
			`UPDATE orders SET completed_at = CURRENT_TIMESTAMP WHERE order_id = ?`,
			orderID,
		)
	}

	return nil
}

func (r *OrderRepository) UpdateTrackingDetail(orderID string, detail string) error {
	query := `UPDATE orders SET tracking_detail = ? WHERE order_id = ?`
	_, err := r.DB.Exec(query, detail, orderID)
	return err
}

func (r *OrderRepository) UpdatePaymentStatus(orderID string, fileName string, email string) error {
	query := `UPDATE orders SET payment_proof = ?, email = ?, status = 'Diproses' WHERE order_id = ?`
	_, err := r.DB.Exec(query, fileName, email, orderID)
	return err
}

func (r *OrderRepository) AjukanPembatalan(orderID string, bank, account, name string) error {
	query := `
        UPDATE orders 
        SET status = 'Pengajuan Pembatalan', 
            refund_bank = ?, 
            refund_account = ?, 
            refund_name = ? 
        WHERE order_id = ?`
	_, err := r.DB.Exec(query, bank, account, name, orderID)
	return err
}

func (r *OrderRepository) BatalkanPesanan(orderID string) error {
	items, err := r.getOrderItems(orderID)
	if err != nil {
		return err
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	for _, item := range items {
		updateStockQuery := `
            UPDATE products 
            SET stock = stock + ?, 
                sold = sold - ? 
            WHERE id = ?`
		if _, err := tx.Exec(updateStockQuery, item.Quantity, item.Quantity, item.ProductID); err != nil {
			tx.Rollback()
			return fmt.Errorf("gagal mengembalikan stok produk ID %d: %v", item.ProductID, err)
		}
	}

	queryUpdateStatus := `UPDATE orders SET status = 'Dibatalkan' WHERE order_id = ?`
	if _, err := tx.Exec(queryUpdateStatus, orderID); err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal update status order: %v", err)
	}

	return tx.Commit()
}

func (r *OrderRepository) getOrderItems(orderID string) ([]OrderItem, error) {
	query := `
        SELECT 
            oi.product_id, 
            p.name, 
            COALESCE(p.image, ''), 
            oi.quantity, 
            oi.price_at_purchase,
            oi.is_reviewed
        FROM order_items oi
        JOIN products p ON oi.product_id = p.id
        WHERE oi.order_id = ?`

	rows, err := r.DB.Query(query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var i OrderItem
		if err := rows.Scan(&i.ProductID, &i.ProductName, &i.Image, &i.Quantity, &i.Price, &i.IsReviewed); err == nil {
			items = append(items, i)
		}
	}
	return items, nil
}

func (r *OrderRepository) MarkItemAsReviewed(orderID string, productID int) error {
	query := `UPDATE order_items SET is_reviewed = 1 WHERE order_id = ? AND product_id = ?`
	_, err := r.DB.Exec(query, orderID, productID)
	return err
}

func (r *OrderRepository) GetUserOrdersByStatus(userID int, status string) ([]Order, error) {
	query := `
        SELECT 
            o.order_id, o.created_at, COALESCE(o.completed_at, ''), o.total_price, o.shipping_cost, o.status, 
            o.buyer_name, o.address, o.payment_method, COALESCE(o.payment_proof, ''),
            COALESCE(o.email, ''), COALESCE(o.phone, ''),
            COALESCE(o.refund_bank, ''), COALESCE(o.refund_account, ''), COALESCE(o.refund_name, ''),
            COALESCE(u.profile_picture, ''), COALESCE(o.tracking_detail, '')
        FROM orders o
        LEFT JOIN users u ON o.user_id = u.id
        WHERE o.user_id = ?`

	var args []interface{}
	args = append(args, userID)

	if status != "" && status != "Semua" {
		query += " AND o.status = ?"
		args = append(args, status)
	}

	query += " ORDER BY o.created_at DESC"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var rawTime, rawCompleted string
		err := rows.Scan(
			&o.OrderID, &rawTime, &rawCompleted, &o.Total, &o.ShippingCost, &o.Status,
			&o.BuyerName, &o.Address, &o.PaymentMethod, &o.PaymentProof,
			&o.Email, &o.Phone,
			&o.RefundBank, &o.RefundAccount, &o.RefundName,
			&o.BuyerProfile, &o.TrackingDetail,
		)
		if err == nil {
			o.Date = parseOrderDate(rawTime)
			if rawCompleted != "" {
				o.CompletedAt = parseOrderDate(rawCompleted)
			} else {
				o.CompletedAt = "-"
			}
			items, _ := r.getOrderItems(o.OrderID)
			o.Items = items
			orders = append(orders, o)
		}
	}
	return orders, nil
}

func (r *OrderRepository) CancelExpiredOrders() error {
	queryGetExpired := `
        SELECT order_id FROM orders 
        WHERE status IN ('Diproses', 'Menunggu Pembayaran') 
        AND payment_method = 'transfer'
        AND (payment_proof IS NULL OR payment_proof = '') 
        AND created_at < datetime('now', '-1 day')`

	rows, err := r.DB.Query(queryGetExpired)
	if err != nil {
		return fmt.Errorf("gagal query orders expired: %v", err)
	}
	defer rows.Close()

	var expiredIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			expiredIDs = append(expiredIDs, id)
		}
	}

	for _, id := range expiredIDs {
		err := r.BatalkanPesanan(id)
		if err != nil {
			fmt.Printf("[SYSTEM] Gagal membatalkan pesanan otomatis %s: %v\n", id, err)
		}
	}
	return nil
}

func parseOrderDate(rawTime string) string {
	if rawTime == "" {
		return "-"
	}
	layouts := []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05Z"}
	var t time.Time
	var err error

	for _, layout := range layouts {
		t, err = time.Parse(layout, rawTime)
		if err == nil {
			break
		}
	}

	if err != nil {
		return rawTime
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		return t.Add(7 * time.Hour).Format("02 Jan 2006 15:04") + " WIB"
	}

	return t.In(loc).Format("02 Jan 2006 15:04") + " WIB"
}