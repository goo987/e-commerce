package handler

import (
	"e-commerce/internal/repository"
	"e-commerce/views/public"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserRepo *repository.UserRepository
}

func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	errQuery := r.URL.Query().Get("error")
	successQuery := r.URL.Query().Get("success")

	var successMsg string
	if successQuery == "true" {
		successMsg = "Akun berhasil dibuat! Silakan login."
	}

	public.Register(errQuery, successMsg).Render(r.Context(), w)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		http.Redirect(w, r, "/register?error=Semua data wajib diisi!", http.StatusSeeOther)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Gagal memproses password", http.StatusInternalServerError)
		return
	}

	err = h.UserRepo.Create(repository.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
		Role:     "user",
		IsActive: 1,
	})

	if err != nil {
		http.Redirect(w, r, "/register?error=Username atau Email sudah terdaftar", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/login?success=registered", http.StatusSeeOther)
}

func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	errParam := r.URL.Query().Get("error")
	successParam := r.URL.Query().Get("success")

	var errMsg string
	var successMsg string

	switch errParam {
	case "1":
		errMsg = "Username atau Password salah!"
	case "inactive":
		errMsg = "Akun Anda tidak aktif. Silakan hubungi admin."
	}

	switch successParam {
	case "registered":
		successMsg = "Pendaftaran berhasil! Silakan masuk."
	case "logout":
		successMsg = "Anda telah logout."
	}

	public.Login(errMsg, successMsg).Render(r.Context(), w)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.UserRepo.GetByUsername(username)
	if err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	if user.IsActive == 0 {
		http.Redirect(w, r, "/login?error=inactive", http.StatusSeeOther)
		return
	}

	sessionValue := fmt.Sprintf("%d|%s|%s", user.ID, user.Username, user.Role)
	cookieName := "session_" + user.Role

	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    sessionValue,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600 * 24,
	}

	http.SetCookie(w, cookie)

	switch user.Role {
	case "admin":
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	case "petugas":
		http.Redirect(w, r, "/petugas/dashboard", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	roles := []string{"admin", "petugas", "user"}
	for _, role := range roles {
		cookie := &http.Cookie{
			Name:   "session_" + role,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		}
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/login?success=logout", http.StatusSeeOther)
}