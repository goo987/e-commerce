package repository

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type Product struct {
	ID          int
	Name        string
	Price       int
	Stock       int
	Image       string
	Sold        int
	CartQty     int
	Description string
	Rating      float64
	TotalReview int
	Reviews     []Review
}

type ProductRepository struct {
	DB *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{DB: db}
}

func (r *ProductRepository) GetAll() ([]Product, error) {
	rows, err := r.DB.Query(`
		SELECT id, name, price, stock, COALESCE(image, ''), sold, COALESCE(description, '')
		FROM products
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Image, &p.Sold, &p.Description)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepository) GetByID(id int) (Product, error) {
	var p Product
	err := r.DB.QueryRow(`
		SELECT id, name, price, stock, COALESCE(image, ''), sold, COALESCE(description, '') 
		FROM products 
		WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Image, &p.Sold, &p.Description)

	if err != nil {
		return Product{}, err
	}
	return p, nil
}

func (r *ProductRepository) Create(p *Product) error {
	_, err := r.DB.Exec(`
		INSERT INTO products (name, price, stock, image, sold, description)
		VALUES (?, ?, ?, ?, 0, ?)
	`, p.Name, p.Price, p.Stock, p.Image, p.Description)
	return err
}

func (r *ProductRepository) Update(p *Product) error {
	_, err := r.DB.Exec(`
		UPDATE products
		SET name = ?, price = ?, stock = ?, image = ?, description = ?
		WHERE id = ?
	`, p.Name, p.Price, p.Stock, p.Image, p.Description, p.ID)
	return err
}

func (r *ProductRepository) Delete(id int) error {
	_, err := r.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
	return err
}

func (r *ProductRepository) GetCartBySelectedIDs(userID int, ids []string) ([]Product, error) {
	if len(ids) == 0 {
		return []Product{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	args[0] = userID

	for i, idStr := range ids {
		placeholders[i] = "?"
		id, _ := strconv.Atoi(idStr)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.name, p.price, p.stock, COALESCE(p.image, ''), c.quantity, COALESCE(p.description, '')
		FROM cart c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = ? AND c.product_id IN (%s)`,
		strings.Join(placeholders, ","))

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Image, &p.CartQty, &p.Description)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepository) GetCartByUserID(userID int) ([]Product, error) {
	query := `
		SELECT p.id, p.name, p.price, p.stock, COALESCE(p.image, ''), c.quantity, COALESCE(p.description, '')
		FROM cart c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = ?`

	rows, err := r.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Image, &p.CartQty, &p.Description)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepository) AddToCart(userID, productID, qty int) error {
	query := `
		INSERT INTO cart (user_id, product_id, quantity) 
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, product_id) 
		DO UPDATE SET quantity = quantity + excluded.quantity`

	_, err := r.DB.Exec(query, userID, productID, qty)
	return err
}

func (r *ProductRepository) UpdateCartQty(userID, productID, qty int) error {
	_, err := r.DB.Exec(`UPDATE cart SET quantity = ? WHERE user_id = ? AND product_id = ?`, qty, userID, productID)
	return err
}

func (r *ProductRepository) DeleteFromCart(userID, productID int) error {
	_, err := r.DB.Exec(`DELETE FROM cart WHERE user_id = ? AND product_id = ?`, userID, productID)
	return err
}

func (r *ProductRepository) DeleteBulkFromCart(userID int, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	args[0] = userID

	for i, idStr := range ids {
		placeholders[i] = "?"
		id, _ := strconv.Atoi(idStr)
		args[i+1] = id
	}

	query := fmt.Sprintf(
		"DELETE FROM cart WHERE user_id = ? AND product_id IN (%s)",
		strings.Join(placeholders, ","),
	)

	_, err := r.DB.Exec(query, args...)
	return err
}