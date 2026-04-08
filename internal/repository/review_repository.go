package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type Review struct {
	ID        int
	ProductID int
	UserID    int
	UserName  string
	Rating    int    
	Comment   string
	CreatedAt string
}

type ReviewRepository struct {
	DB *sql.DB
}

func NewReviewRepository(db *sql.DB) *ReviewRepository {
	return &ReviewRepository{DB: db}
}

func (r *ReviewRepository) CreateReview(rev Review) error {
	query := `
		INSERT INTO reviews (product_id, user_id, user_name, rating, comment)
		VALUES (?, ?, ?, ?, ?)`

	_, err := r.DB.Exec(query, 
		rev.ProductID, 
		rev.UserID, 
		rev.UserName, 
		rev.Rating, 
		rev.Comment,
	)
	
	if err != nil {
		return fmt.Errorf("gagal menyimpan ulasan ke database: %v", err)
	}
	return nil
}

func (r *ReviewRepository) GetReviewsByProductID(productID int) ([]Review, error) {
	query := `
		SELECT id, product_id, user_name, rating, comment, created_at 
		FROM reviews 
		WHERE product_id = ? 
		ORDER BY created_at DESC`

	rows, err := r.DB.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("gagal query reviews: %v", err)
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var rev Review
		var rawTime string
		
		err := rows.Scan(&rev.ID, &rev.ProductID, &rev.UserName, &rev.Rating, &rev.Comment, &rawTime)
		if err != nil {
			fmt.Printf("[DEBUG] Scan Review Error: %v\n", err)
		}
		
		rev.CreatedAt = parseReviewDate(rawTime)
		reviews = append(reviews, rev)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (r *ReviewRepository) GetRatingStats(productID int) (float64, int, error) {
	var avgRating float64
	var totalReviews int
	
	query := `
		SELECT COALESCE(AVG(rating), 0), COUNT(id) 
		FROM reviews 
		WHERE product_id = ?`
	
	err := r.DB.QueryRow(query, productID).Scan(&avgRating, &totalReviews)
	if err != nil {
		return 0, 0, fmt.Errorf("gagal ambil statistik rating: %v", err)
	}
	
	return avgRating, totalReviews, nil
}

func parseReviewDate(rawTime string) string {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}

	var t time.Time
	var err error

	for _, layout := range layouts {
		t, err = time.Parse(layout, rawTime)
		if err == nil {
			return t.Format("02 Jan 2006")
		}
	}
	return rawTime
}