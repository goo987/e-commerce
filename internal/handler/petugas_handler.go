package handler

import (
	"e-commerce/internal/repository"
	"e-commerce/views"
	"e-commerce/views/petugas"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type PetugasHandler struct {
	DashboardRepo *repository.DashboardRepository
	ProductRepo   *repository.ProductRepository
	UserRepo      *repository.UserRepository
	OrderRepo     *repository.OrderRepository
}

func NewPetugasHandler(
	repo *repository.DashboardRepository,
	productRepo *repository.ProductRepository,
	userRepo *repository.UserRepository,
	orderRepo *repository.OrderRepository,
) *PetugasHandler {
	return &PetugasHandler{
		DashboardRepo: repo,
		ProductRepo:   productRepo,
		UserRepo:      userRepo,
		OrderRepo:     orderRepo,
	}
}

func (h *PetugasHandler) PetugasDashboard(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	stats, err := h.DashboardRepo.GetDashboardData(startDate, endDate)
	if err != nil {
		http.Error(w, "Gagal mengambil data statistik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = petugas.Dashboard(stats).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Gagal merender halaman dashboard", http.StatusInternalServerError)
	}
}

func (h *PetugasHandler) PetugasLaporan(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "weekly"
	}

	data, err := h.DashboardRepo.GetLaporanDataPaged(period)
	if err != nil {
		http.Error(w, "Gagal memuat data laporan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	petugas.Laporan(data).Render(r.Context(), w)
}

func (h *PetugasHandler) DetailPesananLaporan(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Redirect(w, r, "/petugas/laporan", http.StatusSeeOther)
		return
	}

	order, err := h.OrderRepo.GetOrderByID(orderID)
	if err != nil {
		fmt.Printf("[DEBUG] Petugas DetailPesananLaporan Error: %v\n", err)
		http.Redirect(w, r, "/petugas/laporan", http.StatusSeeOther)
		return
	}

	petugas.DetailPetugas(order).Render(r.Context(), w)
}

func (h *PetugasHandler) PetugasPesanan(w http.ResponseWriter, r *http.Request) {
	orders, err := h.OrderRepo.GetAllOrders()
	if err != nil {
		http.Error(w, "Gagal mengambil data pesanan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	petugas.Pesanan(orders).Render(r.Context(), w)
}

func (h *PetugasHandler) DetailPesanan(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	order, err := h.OrderRepo.GetOrderByID(orderID)
	if err != nil {
		views.Notfound().Render(r.Context(), w)
		return
	}

	petugas.Detail(order).Render(r.Context(), w)
}

func (h *PetugasHandler) SelesaikanPesanan(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	if orderID == "" {
		http.Error(w, "Order ID tidak boleh kosong", http.StatusBadRequest)
		return
	}

	err := h.OrderRepo.UpdateOrderStatus(orderID, "Selesai")
	if err != nil {
		http.Error(w, "Gagal menyelesaikan pesanan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/pesanan/detail/"+orderID, http.StatusSeeOther)
}

func (h *PetugasHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	orderID := r.FormValue("order_id")
	status := r.FormValue("status")

	if orderID == "" || status == "" {
		http.Error(w, "Order ID atau Status tidak boleh kosong", http.StatusBadRequest)
		return
	}

	err := h.OrderRepo.UpdateOrderStatus(orderID, status)
	if err != nil {
		http.Error(w, "Gagal memperbarui status pesanan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/pesanan/detail/"+orderID, http.StatusSeeOther)
}

func (h *PetugasHandler) UpdateTrackingStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	orderID := r.FormValue("order_id")

	if orderID == "" {
		http.Error(w, "Order ID tidak boleh kosong", http.StatusBadRequest)
		return
	}

	err := h.OrderRepo.UpdateTrackingStep(orderID)
	if err != nil {
		http.Error(w, "Gagal update tracking: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/pesanan/detail/"+orderID, http.StatusSeeOther)
}

func (h *PetugasHandler) KonfirmasiPembatalan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	orderID := r.FormValue("order_id")
	action := r.FormValue("action")

	if orderID == "" {
		http.Error(w, "Order ID tidak boleh kosong", http.StatusBadRequest)
		return
	}

	var err error
	if action == "approve" {
		err = h.OrderRepo.BatalkanPesanan(orderID)
	} else {
		err = h.OrderRepo.UpdateOrderStatus(orderID, "Diproses")
	}

	if err != nil {
		fmt.Printf("[DEBUG] KonfirmasiPembatalan Error: %v\n", err)
		http.Error(w, "Gagal memproses pembatalan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/pesanan/detail/"+orderID, http.StatusSeeOther)
}

func (h *PetugasHandler) PetugasUser(w http.ResponseWriter, r *http.Request) {
	userList, err := h.UserRepo.GetAllUser()
	if err != nil {
		http.Error(w, "Gagal mengambil data user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	petugas.User(userList).Render(r.Context(), w)
}

func (h *PetugasHandler) ToggleUserStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id == 0 {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	status, _ := strconv.Atoi(r.URL.Query().Get("status"))

	newStatus := 1
	if status == 1 {
		newStatus = 0
	}

	err = h.UserRepo.SetActiveStatus(id, newStatus)
	if err != nil {
		http.Error(w, "Gagal memperbarui status user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/user", http.StatusSeeOther)
}

func (h *PetugasHandler) PetugasProduk(w http.ResponseWriter, r *http.Request) {
	products, err := h.ProductRepo.GetAll()
	if err != nil {
		http.Error(w, "Gagal mengambil produk: "+err.Error(), http.StatusInternalServerError)
		return
	}

	petugas.Produk(products).Render(r.Context(), w)
}

func (h *PetugasHandler) CreateProduk(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Gagal parsing form", http.StatusBadRequest)
		return
	}

	price, _ := strconv.Atoi(r.FormValue("price"))
	stock, _ := strconv.Atoi(r.FormValue("stock"))

	file, handlerFile, err := r.FormFile("image")
	var fileName string

	if err == nil {
		defer file.Close()
		fileName = handlerFile.Filename
		os.MkdirAll("./image", os.ModePerm)
		dst, err := os.Create("./image/" + fileName)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)
		}
	}

	err = h.ProductRepo.Create(&repository.Product{
		Name:        r.FormValue("name"),
		Price:       price,
		Stock:       stock,
		Image:       fileName,
		Description: r.FormValue("description"),
		Sold:        0,
	})
	if err != nil {
		http.Error(w, "Gagal membuat produk", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/produk", http.StatusSeeOther)
}

func (h *PetugasHandler) UpdateProduk(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Gagal parsing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	price, _ := strconv.Atoi(r.FormValue("price"))
	stock, _ := strconv.Atoi(r.FormValue("stock"))
	oldImage := r.FormValue("old_image")

	file, handlerFile, err := r.FormFile("image")
	var fileName string

	if err != nil {
		fileName = oldImage
	} else {
		defer file.Close()
		fileName = handlerFile.Filename
		os.MkdirAll("./image", os.ModePerm)
		dst, err := os.Create("./image/" + fileName)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)
		}
	}

	err = h.ProductRepo.Update(&repository.Product{
		ID:          id,
		Name:        r.FormValue("name"),
		Price:       price,
		Stock:       stock,
		Image:       fileName,
		Description: r.FormValue("description"),
	})
	if err != nil {
		http.Error(w, "Gagal update produk", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/produk", http.StatusSeeOther)
}

func (h *PetugasHandler) DeleteProduk(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	err = h.ProductRepo.Delete(id)
	if err != nil {
		http.Error(w, "Gagal menghapus produk", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/petugas/produk", http.StatusSeeOther)
}