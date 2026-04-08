package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SalesData struct {
	DayLabel      string
	Count         int
	HeightPercent int
}

type DashboardStats struct {
	TotalPengunjung int64
	TotalProfit     int64
	TotalPesanan    int64
	TotalProduk     int64
	ProdukTerjual   int64
	ProdukTerlaris  []Product
	RecentOrders    []Order
	SalesChart      []SalesData
}

type StokProduk struct {
	Name        string
	Category    string
	StokMasuk   int
	StokTerjual int
	StokTersisa int
	Image       string
}

type LaporanData struct {
	Transaksi    []Order
	Stok         []StokProduk
	SalesChart   []SalesData
	ProdukTop    []Product
	TotalTerjual int
}

type BackupRecord struct {
	Date     string
	DataType string
	Status   string
}

type DashboardRepository struct {
	DB *sql.DB
}

func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{DB: db}
}

func (r *DashboardRepository) IncrementVisitor() error {
	query := `
        INSERT INTO visitor_stats (date, count) 
        VALUES (DATE('now','localtime'), 1) 
        ON CONFLICT(date) DO UPDATE SET count = count + 1`
	_, err := r.DB.Exec(query)
	return err
}

func (r *DashboardRepository) GetDashboardData(startDate, endDate string) (DashboardStats, error) {
	var stats DashboardStats

	_ = r.DB.QueryRow("SELECT COUNT(id) FROM products").Scan(&stats.TotalProduk)
	_ = r.DB.QueryRow("SELECT COALESCE(SUM(count), 0) FROM visitor_stats").Scan(&stats.TotalPengunjung)

	whereClause := ""
	var args []interface{}
	if startDate != "" && endDate != "" {
		whereClause = " WHERE date(created_at,'localtime') BETWEEN ? AND ?"
		args = append(args, startDate, endDate)
	}

	_ = r.DB.QueryRow("SELECT COUNT(id) FROM orders"+whereClause, args...).Scan(&stats.TotalPesanan)
	_ = r.DB.QueryRow("SELECT COALESCE(SUM(total_price), 0) FROM orders"+whereClause, args...).Scan(&stats.TotalProfit)

	querySold := `
        SELECT COALESCE(SUM(oi.quantity), 0) 
        FROM order_items oi 
        JOIN orders o ON oi.order_id = o.order_id` + whereClause
	_ = r.DB.QueryRow(querySold, args...).Scan(&stats.ProdukTerjual)

	rowsTerlaris, err := r.DB.Query(`
        SELECT p.id, p.name, p.price, p.stock, COALESCE(p.image, ''), IFNULL(SUM(oi.quantity), 0) as total_sold
        FROM products p
        LEFT JOIN order_items oi ON p.id = oi.product_id
        GROUP BY p.id
        ORDER BY total_sold DESC
        LIMIT 4`)
	if err == nil {
		defer rowsTerlaris.Close()
		for rowsTerlaris.Next() {
			var p Product
			if err := rowsTerlaris.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Image, &p.Sold); err == nil {
				stats.ProdukTerlaris = append(stats.ProdukTerlaris, p)
			}
		}
	}

	var rowsOrders *sql.Rows
	if startDate != "" && endDate != "" {
		queryOrders := `
            SELECT order_id, strftime('%d-%m-%Y', created_at), buyer_name, total_price, status 
            FROM orders 
            WHERE date(created_at,'localtime') BETWEEN ? AND ? 
            ORDER BY created_at DESC`
		rowsOrders, err = r.DB.Query(queryOrders, startDate, endDate)
	} else {
		queryOrders := `
            SELECT order_id, strftime('%d-%m-%Y', created_at), buyer_name, total_price, status 
            FROM orders 
            ORDER BY created_at DESC 
            LIMIT 5`
		rowsOrders, err = r.DB.Query(queryOrders)
	}

	if err == nil {
		defer rowsOrders.Close()
		for rowsOrders.Next() {
			var o Order
			if err := rowsOrders.Scan(&o.OrderID, &o.Date, &o.BuyerName, &o.Total, &o.Status); err == nil {
				stats.RecentOrders = append(stats.RecentOrders, o)
			}
		}
	}

	stats.SalesChart = r.GetSalesChartData("weekly")
	return stats, nil
}

func (r *DashboardRepository) GetLaporanData() (LaporanData, error) {
	return r.GetLaporanDataPaged("weekly")
}

func (r *DashboardRepository) GetLaporanDataPaged(period string) (LaporanData, error) {
	var data LaporanData

	rowsT, err := r.DB.Query(`SELECT order_id, strftime('%d-%m-%Y', created_at), buyer_name, total_price, status FROM orders ORDER BY created_at DESC`)
	if err == nil {
		defer rowsT.Close()
		for rowsT.Next() {
			var o Order
			if err := rowsT.Scan(&o.OrderID, &o.Date, &o.BuyerName, &o.Total, &o.Status); err == nil {
				data.Transaksi = append(data.Transaksi, o)
			}
		}
	}

	rowsS, err := r.DB.Query(`
        SELECT 
            p.name, 
            'Umum' as category, 
            (p.stock + IFNULL(sold_data.total_qty, 0)) as stok_awal, 
            IFNULL(sold_data.total_qty, 0) as terjual, 
            p.stock, 
            COALESCE(p.image, '')
        FROM products p
        LEFT JOIN (
            SELECT product_id, SUM(quantity) as total_qty 
            FROM order_items 
            GROUP BY product_id
        ) sold_data ON p.id = sold_data.product_id`)

	if err == nil {
		defer rowsS.Close()
		for rowsS.Next() {
			var s StokProduk
			if err := rowsS.Scan(&s.Name, &s.Category, &s.StokMasuk, &s.StokTerjual, &s.StokTersisa, &s.Image); err == nil {
				data.Stok = append(data.Stok, s)
			}
		}
	}

	data.SalesChart = r.GetSalesChartData(period)

	rowsTop, err := r.DB.Query(`
        SELECT p.id, p.name, COALESCE(p.image, ''), IFNULL(SUM(oi.quantity), 0) as total_sold
        FROM products p
        JOIN order_items oi ON p.id = oi.product_id
        GROUP BY p.id ORDER BY total_sold DESC LIMIT 6`)
	if err == nil {
		defer rowsTop.Close()
		for rowsTop.Next() {
			var p Product
			if err := rowsTop.Scan(&p.ID, &p.Name, &p.Image, &p.Sold); err == nil {
				data.ProdukTop = append(data.ProdukTop, p)
				data.TotalTerjual += int(p.Sold)
			}
		}
	}

	return data, nil
}

func (r *DashboardRepository) GetSalesChartData(period string) []SalesData {
	var chart []SalesData
	var query string

	switch period {
	case "monthly":
		query = `
            WITH RECURSIVE dates(date) AS (
                SELECT date('now','localtime','-29 days')
                UNION ALL
                SELECT date(date, '+1 day') FROM dates WHERE date < date('now','localtime')
            )
            SELECT strftime('%d/%m', d.date) as label, COUNT(o.id) as total
            FROM dates d
            LEFT JOIN orders o ON date(o.created_at,'localtime') = d.date
            GROUP BY d.date ORDER BY d.date ASC`

	case "yearly":
	query = `
        WITH RECURSIVE months(m) AS (
            SELECT 1
            UNION ALL
            SELECT m + 1 FROM months WHERE m < 12
        )
        SELECT 
            CASE m 
                WHEN 1 THEN 'JAN' WHEN 2 THEN 'FEB' WHEN 3 THEN 'MAR' WHEN 4 THEN 'APR'
                WHEN 5 THEN 'MEI' WHEN 6 THEN 'JUN' WHEN 7 THEN 'JUL' WHEN 8 THEN 'AGU'
                WHEN 9 THEN 'SEP' WHEN 10 THEN 'OKT' WHEN 11 THEN 'NOV' WHEN 12 THEN 'DES'
            END as label,
            COUNT(o.id) as total
        FROM months
        LEFT JOIN orders o 
            ON CAST(strftime('%m', o.created_at,'localtime') AS INTEGER) = m 
            AND strftime('%Y', o.created_at,'localtime') = strftime('%Y', 'now','localtime')
        GROUP BY m 
        ORDER BY m ASC`

	default:
		query = `
            WITH RECURSIVE dates(date) AS (
                SELECT date('now','localtime','weekday 0','-6 days')
                UNION ALL
                SELECT date(date, '+1 day') FROM dates WHERE date < date('now','localtime','weekday 0')
            )
            SELECT 
                (CASE strftime('%w', d.date) 
                    WHEN '0' THEN 'MIN' WHEN '1' THEN 'SEN' WHEN '2' THEN 'SEL' 
                    WHEN '3' THEN 'RAB' WHEN '4' THEN 'KAM' WHEN '5' THEN 'JUM' 
                    WHEN '6' THEN 'SAB' END) as label,
                COUNT(o.id) as total
            FROM dates d
            LEFT JOIN orders o ON date(o.created_at,'localtime') = d.date
            GROUP BY d.date ORDER BY d.date ASC`
	}

	rows, err := r.DB.Query(query)
	if err != nil {
		return chart
	}
	defer rows.Close()

	maxCount := 0
	for rows.Next() {
		var s SalesData
		if err := rows.Scan(&s.DayLabel, &s.Count); err == nil {
			if s.Count > maxCount {
				maxCount = s.Count
			}
			chart = append(chart, s)
		}
	}

	for i := range chart {
		if maxCount > 0 {
			chart[i].HeightPercent = (chart[i].Count * 100) / maxCount
			if chart[i].HeightPercent < 15 && chart[i].Count > 0 {
				chart[i].HeightPercent = 15
			}
		} else {
			chart[i].HeightPercent = 0
		}
	}

	return chart
}

func (r *DashboardRepository) GetWeeklyChartData() []SalesData {
	return r.GetSalesChartData("weekly")
}

func (r *DashboardRepository) GetBackupHistory() ([]BackupRecord, error) {
	query := `SELECT date, data_type, status FROM backup_records ORDER BY id DESC`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []BackupRecord
	for rows.Next() {
		var b BackupRecord
		if err := rows.Scan(&b.Date, &b.DataType, &b.Status); err == nil {
			history = append(history, b)
		}
	}
	return history, nil
}

func (r *DashboardRepository) LogBackup(dataType, status string) error {
	now := time.Now().Format("02 Jan 2006, 15:04:05")
	query := `INSERT INTO backup_records (date, data_type, status) VALUES (?, ?, ?)`
	_, err := r.DB.Exec(query, now, dataType, status)
	return err
}

func (r *DashboardRepository) GetSQLDump() (string, error) {
	var dump strings.Builder
	dump.WriteString("-- Hostel Mart Database Backup\n")
	dump.WriteString(fmt.Sprintf("-- Generated at: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	rows, err := r.DB.Query("SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, createSQL string
		if err := rows.Scan(&tableName, &createSQL); err != nil {
			return "", err
		}

		dump.WriteString(fmt.Sprintf("-- Structure for: %s\nDROP TABLE IF EXISTS %s;\n%s;\n\n", tableName, tableName, createSQL))

		dataRows, err := r.DB.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
		if err != nil {
			return "", err
		}

		cols, _ := dataRows.Columns()
		for dataRows.Next() {
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := dataRows.Scan(columnPointers...); err != nil {
				continue
			}

			var values []string
			for _, col := range columns {
				if col == nil {
					values = append(values, "NULL")
				} else {
					var valStr string
					switch v := col.(type) {
					case []byte:
						valStr = string(v)
					default:
						valStr = fmt.Sprintf("%v", v)
					}
					escaped := strings.ReplaceAll(valStr, "'", "''")
					values = append(values, fmt.Sprintf("'%s'", escaped))
				}
			}
			dump.WriteString(fmt.Sprintf("INSERT INTO %s VALUES (%s);\n", tableName, strings.Join(values, ",")))
		}
		dataRows.Close()
		dump.WriteString("\n")
	}

	return dump.String(), nil
}