package repository

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int
	Username       string
	Email          string
	Password       string
	Role           string
	Phone          string
	ProfilePicture string
	IsActive       int
	CreatedAt      time.Time
}

type UserRepository struct {
	DB *sql.DB
}

func (r *UserRepository) EnsureAdminExists() {
	var exists bool
	r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')").Scan(&exists)

	if !exists {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
		query := `
		INSERT INTO users (username, email, password, role, is_active, created_at)
		VALUES (?, ?, ?, 'admin', 1, (datetime('now','localtime')))`

		r.DB.Exec(query, "admin", "admin@mail.com", string(hashedPassword))
		fmt.Println("Admin default created: admin / admin123")
	}
}

func (r *UserRepository) SetActiveStatus(id int, active int) error {
	query := `UPDATE users SET is_active = ? WHERE id = ?`
	_, err := r.DB.Exec(query, active, id)
	return err
}

func (r *UserRepository) GetByID(id int) (*User, error) {
	query := `
	SELECT id, username, email, password, role, IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
	FROM users
	WHERE id = ?`

	row := r.DB.QueryRow(query, id)

	var user User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.Phone,
		&user.ProfilePicture,
		&user.IsActive,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*User, error) {
	query := `
	SELECT id, username, email, password, role, IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
	FROM users
	WHERE username = ?`

	row := r.DB.QueryRow(query, username)

	var user User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.Phone,
		&user.ProfilePicture,
		&user.IsActive,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user User) error {
	query := `
	INSERT INTO users (username, email, password, role, is_active, created_at, phone, profile_picture)
	VALUES (?, ?, ?, ?, 1, (datetime('now','localtime')), ?, ?)`

	_, err := r.DB.Exec(query,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.Phone,
		user.ProfilePicture,
	)
	return err
}

func (r *UserRepository) CreatePetugas(username, email, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO users (username, email, password, role, is_active, created_at)
	VALUES (?, ?, ?, 'petugas', 1, (datetime('now','localtime')))`

	_, err = r.DB.Exec(query, username, email, string(hashedPassword))
	return err
}

func (r *UserRepository) GetAllPetugas() ([]User, error) {
	query := `
	SELECT id, username, email, password, role, IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
	FROM users
	WHERE role = 'petugas'
	ORDER BY id DESC`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Password,
			&user.Role,
			&user.Phone,
			&user.ProfilePicture,
			&user.IsActive,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepository) UpdatePetugas(id int, username, email string) error {
	query := `UPDATE users SET username = ?, email = ? WHERE id = ? AND role = 'petugas'`
	_, err := r.DB.Exec(query, username, email, id)
	return err
}

func (r *UserRepository) DeletePetugas(id int) error {
	query := `DELETE FROM users WHERE id = ? AND role = 'petugas'`
	_, err := r.DB.Exec(query, id)
	return err
}

func (r *UserRepository) GetAllUser() ([]User, error) {
	query := `
	SELECT id, username, email, password, role, IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
	FROM users
	WHERE role = 'user'
	ORDER BY id DESC`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Password,
			&user.Role,
			&user.Phone,
			&user.ProfilePicture,
			&user.IsActive,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepository) UpdateUserField(id int, field string, value string) error {
	allowedFields := map[string]bool{
		"username": true,
		"email":    true,
		"phone":    true,
	}

	if !allowedFields[field] {
		return fmt.Errorf("field %s tidak valid", field)
	}

	query := fmt.Sprintf("UPDATE users SET %s = ? WHERE id = ?", field)
	_, err := r.DB.Exec(query, value, id)
	return err
}

func (r *UserRepository) UpdatePassword(id int, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}

	query := `UPDATE users SET password = ? WHERE id = ?`
	_, err = r.DB.Exec(query, string(hashedPassword), id)
	return err
}

func (r *UserRepository) UpdateProfilePicture(id int, filePath string) error {
	query := `UPDATE users SET profile_picture = ? WHERE id = ?`
	_, err := r.DB.Exec(query, filePath, id)
	return err
}

func (r *UserRepository) AddAddress(userID int, label, address string) error {
	var count int
	r.DB.QueryRow("SELECT COUNT(*) FROM addresses WHERE user_id = ?", userID).Scan(&count)

	if count >= 3 {
		return fmt.Errorf("maksimal 3 alamat")
	}

	var hasDefault int
	r.DB.QueryRow("SELECT COUNT(*) FROM addresses WHERE user_id = ? AND is_default = 1", userID).Scan(&hasDefault)

	isDefault := 0
	if hasDefault == 0 {
		isDefault = 1
	}

	query := `
	INSERT INTO addresses (user_id, label, address, is_default)
	VALUES (?, ?, ?, ?)`

	_, err := r.DB.Exec(query, userID, label, address, isDefault)
	return err
}

func (r *UserRepository) UpdateAddress(userID, id int, label, address string) error {
	query := `
		UPDATE addresses 
		SET label = ?, address = ?
		WHERE id = ? AND user_id = ?
	`
	_, err := r.DB.Exec(query, label, address, id, userID)
	return err
}

func (r *UserRepository) DeleteAddress(userID, addressID int) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	var isDefault int
	err = tx.QueryRow("SELECT is_default FROM addresses WHERE id = ? AND user_id = ?", addressID, userID).Scan(&isDefault)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM addresses WHERE id = ? AND user_id = ?", addressID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if isDefault == 1 {
		var nextID int
		err = tx.QueryRow("SELECT id FROM addresses WHERE user_id = ? LIMIT 1", userID).Scan(&nextID)
		if err == nil {
			tx.Exec("UPDATE addresses SET is_default = 1 WHERE id = ?", nextID)
		}
	}

	return tx.Commit()
}

func (r *UserRepository) SetDefaultAddress(userID, addressID int) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	tx.Exec("UPDATE addresses SET is_default = 0 WHERE user_id = ?", userID)

	_, err = tx.Exec("UPDATE addresses SET is_default = 1 WHERE id = ? AND user_id = ?", addressID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *UserRepository) GetDefaultAddress(userID int) (string, error) {
	query := `
	SELECT address 
	FROM addresses 
	WHERE user_id = ? AND is_default = 1 
	LIMIT 1`

	var address string
	err := r.DB.QueryRow(query, userID).Scan(&address)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Alamat belum diatur", nil
		}
		return "", err
	}
	return address, nil
}

func (r *UserRepository) GetUserAddresses(userID int) ([]map[string]interface{}, error) {
	query := `
	SELECT id, label, address, is_default 
	FROM addresses 
	WHERE user_id = ?
	ORDER BY is_default DESC, id DESC`

	rows, err := r.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int
		var label, address string
		var isDefault int

		err := rows.Scan(&id, &label, &address, &isDefault)
		if err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"id":         id,
			"label":      label,
			"address":    address,
			"is_default": isDefault == 1,
		})
	}
	return results, nil
}