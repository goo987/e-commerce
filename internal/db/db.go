package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	dsn := "./tmp/app.db?_busy_timeout=5000&_foreign_keys=on&_journal=WAL"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		email TEXT,
		password TEXT,
		role TEXT DEFAULT 'user',
		is_active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, 
		alamat TEXT,
		phone TEXT,
		profile_picture TEXT
	);

	CREATE TABLE IF NOT EXISTS addresses (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER,
	label TEXT,
	address TEXT,
	is_default INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price INTEGER NOT NULL,
		stock INTEGER DEFAULT 0,
		image TEXT,
		sold INTEGER DEFAULT 0,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id TEXT UNIQUE,
		user_id INTEGER,
		buyer_name TEXT,
		email TEXT,
		phone TEXT,
		address TEXT,
		total_price INTEGER,
		shipping_cost INTEGER DEFAULT 20000,
		payment_method TEXT,
		payment_proof TEXT,
		status TEXT DEFAULT 'Diproses',
		receipt_number TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		refund_bank TEXT,
		refund_account TEXT,
		refund_name TEXT,
		tracking_detail TEXT,
		tracking_step INTEGER DEFAULT 1,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id TEXT,
		product_id INTEGER,
		quantity INTEGER,
		price_at_purchase INTEGER,
		is_reviewed INTEGER DEFAULT 0,
		FOREIGN KEY (order_id) REFERENCES orders(order_id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);

	CREATE TABLE IF NOT EXISTS visitor_stats (
		date DATE PRIMARY KEY,
		count INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS cart (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		product_id INTEGER,
		quantity INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (product_id) REFERENCES products(id),
		UNIQUE(user_id, product_id)
	);

	CREATE TABLE IF NOT EXISTS backup_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT,
    data_type TEXT,
    status TEXT
	);

	CREATE TABLE IF NOT EXISTS reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    user_name TEXT,
    rating INTEGER CHECK(rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("gagal migrasi database: %v", err)
	}

	return db, nil
}
