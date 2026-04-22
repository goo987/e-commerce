package handler

import (
	"e-commerce/internal/middleware"
	"e-commerce/internal/repository"
	"e-commerce/views/public"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type PublicHandler struct {
	ProductRepo *repository.ProductRepository
	UserRepo    *repository.UserRepository
	OrderRepo   *repository.OrderRepository
	ReviewRepo  *repository.ReviewRepository
}

func NewPublicHandler(productRepo *repository.ProductRepository, userRepo *repository.UserRepository, orderRepo *repository.OrderRepository, reviewRepo *repository.ReviewRepository) *PublicHandler {
	return &PublicHandler{
		ProductRepo: productRepo,
		UserRepo:    userRepo,
		OrderRepo:   orderRepo,
		ReviewRepo:  reviewRepo,
	}
}

func (h *PublicHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	label := r.FormValue("label")
	address := r.FormValue("address")

	if address == "" {
		sendJSONError(w, "Alamat tidak boleh kosong", http.StatusBadRequest)
		return
	}

	err := h.UserRepo.AddAddress(userID, label, address)
	if err != nil {
		sendJSONError(w, "Gagal menambah alamat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.FormValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendJSONError(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	err = h.UserRepo.DeleteAddress(userID, id)
	if err != nil {
		sendJSONError(w, "Gagal hapus alamat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) SetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.FormValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendJSONError(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	err = h.UserRepo.SetDefaultAddress(userID, id)
	if err != nil {
		sendJSONError(w, "Gagal set default", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.FormValue("id")
	label := r.FormValue("label")
	address := r.FormValue("address")

	if label == "" || address == "" {
		http.Error(w, "Data tidak lengkap", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	err = h.UserRepo.UpdateAddress(userID, id, label, address)
	if err != nil {
		fmt.Println("UpdateAddress error:", err)
		http.Error(w, "Gagal update alamat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) Akun(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.UserRepo.GetByID(userID)
	if err != nil {
		fmt.Printf("[DEBUG] Akun Error: %v\n", err)
		http.Error(w, "Gagal memuat profil", http.StatusInternalServerError)
		return
	}

	addresses, err := h.UserRepo.GetUserAddresses(userID)
	if err != nil {
		fmt.Printf("[DEBUG] Address Error: %v\n", err)
		addresses = []map[string]interface{}{}
	}

	public.Akun(*user, addresses).Render(r.Context(), w)
}

func (h *PublicHandler) UpdateAkun(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fieldName := r.FormValue("field")
	newValue := r.FormValue("value")

	if fieldName == "phone" {
		if newValue == "" {
			sendJSONError(w, "Nomor HP tidak boleh kosong", http.StatusBadRequest)
			return
		}
		for _, c := range newValue {
			if c < '0' || c > '9' {
				sendJSONError(w, "Nomor HP harus berupa angka", http.StatusBadRequest)
				return
			}
		}
		if len(newValue) < 10 || len(newValue) > 13 {
			sendJSONError(w, "Nomor HP harus 10-13 digit", http.StatusBadRequest)
			return
		}
	}

	err := h.UserRepo.UpdateUserField(userID, fieldName, newValue)
	if err != nil {
		sendJSONError(w, "Gagal memperbarui data", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newPassword := r.FormValue("new_password")
	if len(newPassword) < 6 {
		sendJSONError(w, "Password minimal 6 karakter", http.StatusBadRequest)
		return
	}

	err := h.UserRepo.UpdatePassword(userID, newPassword)
	if err != nil {
		sendJSONError(w, "Gagal ganti password", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) OrderHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	statusParam := r.URL.Query().Get("status")
	if statusParam == "" {
		statusParam = "Semua"
	}

	searchStatus := statusParam
	if searchStatus == "Semua" {
		searchStatus = ""
	}

	orders, err := h.OrderRepo.GetUserOrdersByStatus(userID, searchStatus)
	if err != nil {
		orders = []repository.Order{}
	}
	public.Riwayat(orders, statusParam).Render(r.Context(), w)
}

func (h *PublicHandler) OrderDetail(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	order, err := h.OrderRepo.GetOrderByID(orderID)
	if err != nil {
		fmt.Printf("[DEBUG] OrderDetail Error: %v\n", err)
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	public.Detail(order).Render(r.Context(), w)
}

func (h *PublicHandler) Home(w http.ResponseWriter, r *http.Request) {
	products, err := h.ProductRepo.GetAll()
	if err != nil {
		http.Error(w, "Gagal memuat produk", http.StatusInternalServerError)
		return
	}

	userID, isLoggedIn := r.Context().Value(middleware.UserIDKey).(int)

	cartCount := 0
	if isLoggedIn {
		cartCount, _ = h.ProductRepo.GetCartCount(userID)
	}

	public.Homepage(products, isLoggedIn, cartCount).Render(r.Context(), w)
}

func (h *PublicHandler) ProductDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)
	product, err := h.ProductRepo.GetByID(id)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	avg, count, errStats := h.ReviewRepo.GetRatingStats(id)
	if errStats != nil {
		fmt.Printf("[DEBUG] GetRatingStats Error: %v\n", errStats)
	}

	reviews, errRev := h.ReviewRepo.GetReviewsByProductID(id)
	if errRev != nil {
		fmt.Printf("[DEBUG] GetReviews Error: %v\n", errRev)
		reviews = []repository.Review{}
	}

	product.Rating = avg
	product.TotalReview = count
	product.Reviews = reviews

	shippingCost := 20000

	_, isLoggedIn := r.Context().Value(middleware.UserIDKey).(int)

	public.ProdukDetail(product, shippingCost, isLoggedIn).Render(r.Context(), w)
}

func (h *PublicHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	productID, _ := strconv.Atoi(r.FormValue("product_id"))
	qty, _ := strconv.Atoi(r.FormValue("qty"))

	if qty <= 0 {
		qty = 1
	}

	err := h.ProductRepo.AddToCart(userID, productID, qty)
	if err != nil {
		fmt.Printf("[DEBUG] AddToCart Error: %v\n", err)
	}
	http.Redirect(w, r, "/keranjang", http.StatusSeeOther)
}

func (h *PublicHandler) Cart(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	products, err := h.ProductRepo.GetCartByUserID(userID)
	if err != nil {
		products = []repository.Product{}
	}
	public.Keranjang(products).Render(r.Context(), w)
}

func (h *PublicHandler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	productID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	h.ProductRepo.DeleteFromCart(userID, productID)
	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) UpdateCartQty(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	productID, _ := strconv.Atoi(r.URL.Query().Get("product_id"))
	newQty, _ := strconv.Atoi(r.URL.Query().Get("qty"))

	if newQty <= 0 {
		h.ProductRepo.DeleteFromCart(userID, productID)
	} else {
		h.ProductRepo.UpdateCartQty(userID, productID, newQty)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idsParam := r.URL.Query().Get("ids")
	user, _ := h.UserRepo.GetByID(userID)

	addresses, _ := h.UserRepo.GetUserAddresses(userID)

	selectedIDs := strings.Split(idsParam, ",")
	products, err := h.ProductRepo.GetCartBySelectedIDs(userID, selectedIDs)
	if err != nil || len(products) == 0 {
		http.Redirect(w, r, "/keranjang", http.StatusSeeOther)
		return
	}

	public.Checkout(products, *user, addresses).Render(r.Context(), w)
}

func (h *PublicHandler) ProsesCheckout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Sesi berakhir", http.StatusUnauthorized)
		return
	}

	user, err := h.UserRepo.GetByID(userID)
	if err != nil {
		sendJSONError(w, "User tidak ditemukan", http.StatusInternalServerError)
		return
	}

	addressIDStr := r.FormValue("address_id")

	var selectedAddress string

	if addressIDStr == "" {
		selectedAddress, err = h.UserRepo.GetDefaultAddress(userID)
		if err != nil || selectedAddress == "" {
			sendJSONError(w, "Pilih alamat pengiriman", http.StatusBadRequest)
			return
		}
	} else {
		addressID, err := strconv.Atoi(addressIDStr)
		if err != nil {
			sendJSONError(w, "Alamat tidak valid", http.StatusBadRequest)
			return
		}

		query := `
			SELECT label, address 
			FROM addresses 
			WHERE id = ? AND user_id = ?
		`

		var label, addr string
		err = h.UserRepo.DB.QueryRow(query, addressID, userID).Scan(&label, &addr)
		if err != nil {
			sendJSONError(w, "Alamat tidak ditemukan", http.StatusBadRequest)
			return
		}

		selectedAddress = fmt.Sprintf("%s - %s - %s", label, user.Phone, addr)
	}

	if user.Phone == "" {
		sendJSONError(w, "Nomor HP belum diisi", http.StatusBadRequest)
		return
	}

	idsParam := r.FormValue("selected_ids")
	if idsParam == "" {
		sendJSONError(w, "Tidak ada produk terpilih", http.StatusBadRequest)
		return
	}

	selectedIDs := strings.Split(idsParam, ",")
	cartItems, err := h.ProductRepo.GetCartBySelectedIDs(userID, selectedIDs)
	if err != nil || len(cartItems) == 0 {
		sendJSONError(w, "Keranjang kosong", http.StatusBadRequest)
		return
	}

	var orderItems []repository.OrderItemInput
	subtotal := 0
	shippingCost := 20000

	for _, p := range cartItems {
		subtotal += (p.Price * p.CartQty)

		orderItems = append(orderItems, repository.OrderItemInput{
			ProductID: p.ID,
			Quantity:  p.CartQty,
			Price:     p.Price,
		})
	}

	paymentMethod := r.FormValue("payment_method")
	if paymentMethod == "" {
		sendJSONError(w, "Pilih metode pembayaran", http.StatusBadRequest)
		return
	}

	status := "Menunggu Pembayaran"
	if paymentMethod == "cod" {
		status = "Diproses"
	}

	email := r.FormValue("email")
	if email == "" {
		email = user.Email
	}

	input := repository.CreateOrderInput{
		UserID:        userID,
		BuyerName:     user.Username,
		Address:       selectedAddress,
		Phone:         user.Phone,
		Email:         email,
		PaymentMethod: paymentMethod,
		TotalPrice:    subtotal + shippingCost,
		ShippingCost:  shippingCost,
		Status:        status,
		Items:         orderItems,
	}

	orderID, err := h.OrderRepo.CreateOrder(input)
	if err != nil {
		fmt.Printf("[DEBUG] CreateOrder Error: %v\n", err)
		sendJSONError(w, "Gagal membuat pesanan", http.StatusInternalServerError)
		return
	}

	_ = h.ProductRepo.DeleteBulkFromCart(userID, selectedIDs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"order_id": orderID,
	})
}

func (h *PublicHandler) Pembayaran(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	order, err := h.OrderRepo.GetOrderByID(orderID)
	if err != nil {
		fmt.Printf("[DEBUG] GetOrderByID Error: %v\n", err)
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	if order.PaymentProof != "" {
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	public.Pembayaran(order).Render(r.Context(), w)
}

func (h *PublicHandler) KonfirmasiPembayaran(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		sendJSONError(w, "File terlalu besar", http.StatusBadRequest)
		return
	}

	orderID := r.FormValue("order_id")
	email := r.FormValue("email")

	file, header, err := r.FormFile("payment_proof")
	if err != nil {
		sendJSONError(w, "Bukti pembayaran wajib diunggah", http.StatusBadRequest)
		return
	}
	defer file.Close()

	uploadDir := "bukti_transfer"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadDir, 0755)
		if err != nil {
			sendJSONError(w, "Gagal membuat folder penyimpanan", http.StatusInternalServerError)
			return
		}
	}

	ext := filepath.Ext(header.Filename)
	newFileName := fmt.Sprintf("proof-%s-%d%s", orderID, time.Now().Unix(), ext)
	dstPath := filepath.Join(uploadDir, newFileName)

	dst, err := os.Create(dstPath)
	if err != nil {
		fmt.Printf("[DEBUG] Create File Error: %v\n", err)
		sendJSONError(w, "Gagal menyimpan gambar di server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		sendJSONError(w, "Gagal menyalin isi file", http.StatusInternalServerError)
		return
	}

	err = h.OrderRepo.UpdatePaymentStatus(orderID, newFileName, email)
	if err != nil {
		fmt.Printf("[DEBUG] DB Update Error: %v\n", err)
		sendJSONError(w, "Gagal memperbarui status pesanan di database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Konfirmasi berhasil",
	})
}

func (h *PublicHandler) BulkRemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	h.ProductRepo.DeleteBulkFromCart(userID, ids)
	w.WriteHeader(http.StatusOK)
}

func (h *PublicHandler) BatalkanPesanan(w http.ResponseWriter, r *http.Request) {
	orderID := r.FormValue("order_id")
	bank := r.FormValue("refund_bank")
	account := r.FormValue("refund_account")
	name := r.FormValue("refund_name")

	order, err := h.OrderRepo.GetOrderByID(orderID)
	if err != nil {
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	if order.Status == "Menunggu Pembayaran" {
		err = h.OrderRepo.BatalkanPesanan(orderID)
		if err != nil {
			fmt.Printf("[DEBUG] BatalkanPesanan Error: %v\n", err)
		}
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	if order.Status == "Diproses" {
		if strings.ToLower(order.PaymentMethod) == "cod" {
			err = h.OrderRepo.AjukanPembatalan(orderID, "", "", "")
			if err != nil {
				fmt.Printf("[DEBUG] AjukanPembatalan COD Error: %v\n", err)
			}
			http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
			return
		}

		if bank == "" || account == "" || name == "" {
			http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
			return
		}

		err = h.OrderRepo.AjukanPembatalan(orderID, bank, account, name)
		if err != nil {
			fmt.Printf("[DEBUG] AjukanPembatalan Error: %v\n", err)
		}
		http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/riwayat", http.StatusSeeOther)
}

func (h *PublicHandler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.UserRepo.GetByID(userID)
	if err != nil {
		http.Error(w, "User tidak ditemukan", http.StatusInternalServerError)
		return
	}

	orderID := r.FormValue("order_id")
	productID, _ := strconv.Atoi(r.FormValue("product_id"))
	rating, _ := strconv.Atoi(r.FormValue("rating"))
	comment := r.FormValue("comment")

	newReview := repository.Review{
		ProductID: productID,
		UserID:    userID,
		UserName:  user.Username,
		Rating:    rating,
		Comment:   comment,
	}

	err = h.ReviewRepo.CreateReview(newReview)
	if err != nil {
		fmt.Printf("[DEBUG] CreateReview Error: %v\n", err)
		http.Error(w, "Gagal mengirim ulasan", http.StatusInternalServerError)
		return
	}

	if orderID != "" {
		err = h.OrderRepo.MarkItemAsReviewed(orderID, productID)
		if err != nil {
			fmt.Printf("[DEBUG] MarkItemAsReviewed Error: %v\n", err)
		}
	}

	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

func (h *PublicHandler) UpdateFotoProfil(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		sendJSONError(w, "File terlalu besar", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("profile_picture")
	if err != nil {
		sendJSONError(w, "Gambar tidak ditemukan", http.StatusBadRequest)
		return
	}
	defer file.Close()

	uploadDir := filepath.Join("static", "uploads", "profile")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("[ERROR] Gagal bikin folder: %v\n", err)
		sendJSONError(w, "Server gagal menyiapkan folder penyimpanan", http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(header.Filename)
	newFileName := fmt.Sprintf("user-%d-%d%s", userID, time.Now().Unix(), ext)
	dstPath := filepath.Join(uploadDir, newFileName)

	dst, err := os.Create(dstPath)
	if err != nil {
		fmt.Printf("[ERROR] Gagal buat file: %v\n", err)
		sendJSONError(w, "Gagal membuat file di server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		sendJSONError(w, "Gagal menyimpan isi file", http.StatusInternalServerError)
		return
	}

	dbPath := "/static/uploads/profile/" + newFileName
	err = h.UserRepo.UpdateProfilePicture(userID, dbPath)
	if err != nil {
		sendJSONError(w, "Gagal update database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"filePath": dbPath,
	})
}

func (h *PublicHandler) DeleteFotoProfil(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.UserRepo.GetByID(userID)
	if err == nil && user.ProfilePicture != "" {
		oldPath := "." + user.ProfilePicture
		os.Remove(oldPath)
	}

	err = h.UserRepo.UpdateProfilePicture(userID, "")
	if err != nil {
		sendJSONError(w, "Gagal hapus foto", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": message})
}
