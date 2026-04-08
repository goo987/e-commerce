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
	Alamat         string
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
	SELECT id, username, email, password, role, IFNULL(alamat, ''), IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
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
		&user.Alamat,
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
	SELECT id, username, email, password, role, IFNULL(alamat, ''), IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
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
		&user.Alamat,
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
	INSERT INTO users (username, email, password, role, is_active, created_at, alamat, phone, profile_picture)
	VALUES (?, ?, ?, ?, 1, (datetime('now','localtime')), ?, ?, ?)`

	_, err := r.DB.Exec(query,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.Alamat,
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
	SELECT id, username, email, password, role, IFNULL(alamat, ''), IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
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
			&user.Alamat,
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
	SELECT id, username, email, password, role, IFNULL(alamat, ''), IFNULL(phone, ''), IFNULL(profile_picture, ''), is_active, created_at
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
			&user.Alamat,
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
		"alamat":   true,
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