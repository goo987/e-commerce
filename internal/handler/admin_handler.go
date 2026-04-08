package handler

import (
	"e-commerce/internal/repository"
	"e-commerce/views/admin"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	UserRepo      *repository.UserRepository
	ProductRepo   *repository.ProductRepository
	DashboardRepo *repository.DashboardRepository
	OrderRepo     *repository.OrderRepository
}

// AdminDashboard menangani halaman utama statistik admin
func (h *AdminHandler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	stats, err := h.DashboardRepo.GetDashboardData(startDate, endDate)
	if err != nil {
		http.Error(w, "Gagal memuat data dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	admin.Dashboard(stats).Render(r.Context(), w)
}

func (h *AdminHandler) AdminLaporan(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "weekly"
	}

	data, err := h.DashboardRepo.GetLaporanDataPaged(period)
	if err != nil {
		http.Error(w, "Gagal memuat data laporan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	admin.Laporan(data).Render(r.Context(), w)
}

func (h *AdminHandler) DetailPesananLaporan(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Redirect(w, r, "/admin/laporan", http.StatusSeeOther)
		return
	}

	order, err := h.OrderRepo.GetOrderByID(orderID)
	if err != nil {
		fmt.Printf("[DEBUG] Admin DetailPesanan Error: %v\n", err)
		http.Redirect(w, r, "/admin/laporan", http.StatusSeeOther)
		return
	}

	admin.DetailPesanan(order).Render(r.Context(), w)
}

func (h *AdminHandler) AdminPetugas(w http.ResponseWriter, r *http.Request) {
	petugasList, err := h.UserRepo.GetAllPetugas()
	if err != nil {
		http.Error(w, "Gagal mengambil data petugas", http.StatusInternalServerError)
		return
	}

	admin.Petugas(petugasList).Render(r.Context(), w)
}

func (h *AdminHandler) CreatePetugas(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Gagal parsing form", http.StatusBadRequest)
		return
	}

	err := h.UserRepo.CreatePetugas(
		r.FormValue("username"),
		r.FormValue("email"),
		r.FormValue("password"),
	)
	if err != nil {
		http.Error(w, "Gagal membuat petugas", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/petugas", http.StatusSeeOther)
}

func (h *AdminHandler) UpdatePetugas(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Gagal parsing form", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	err = h.UserRepo.UpdatePetugas(
		id,
		r.FormValue("username"),
		r.FormValue("email"),
	)
	if err != nil {
		http.Error(w, "Gagal update petugas", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/petugas", http.StatusSeeOther)
}

func (h *AdminHandler) TogglePetugasStatus(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Gagal mengubah status petugas", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/petugas", http.StatusSeeOther)
}

func (h *AdminHandler) DeletePetugas(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	err = h.UserRepo.DeletePetugas(id)
	if err != nil {
		http.Error(w, "Gagal menghapus petugas", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/petugas", http.StatusSeeOther)
}

func (h *AdminHandler) AdminUser(w http.ResponseWriter, r *http.Request) {
	userList, err := h.UserRepo.GetAllUser()
	if err != nil {
		http.Error(w, "Gagal mengambil data user", http.StatusInternalServerError)
		return
	}

	admin.User(userList).Render(r.Context(), w)
}

func (h *AdminHandler) ToggleUserStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
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

	http.Redirect(w, r, "/admin/user", http.StatusSeeOther)
}

func (h *AdminHandler) AdminProduk(w http.ResponseWriter, r *http.Request) {
	products, err := h.ProductRepo.GetAll()
	if err != nil {
		http.Error(w, "Gagal mengambil produk: "+err.Error(), http.StatusInternalServerError)
		return
	}

	admin.Produk(products).Render(r.Context(), w)
}

func (h *AdminHandler) CreateProduk(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, "/admin/produk", http.StatusSeeOther)
}

func (h *AdminHandler) UpdateProduk(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, "/admin/produk", http.StatusSeeOther)
}

func (h *AdminHandler) DeleteProduk(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, "/admin/produk", http.StatusSeeOther)
}

func (h *AdminHandler) AdminBackup(w http.ResponseWriter, r *http.Request) {
	history, err := h.DashboardRepo.GetBackupHistory()
	if err != nil {
		fmt.Printf("[DEBUG] Gagal ambil riwayat backup: %v\n", err)
		history = []repository.BackupRecord{}
	}

	admin.Backup(history).Render(r.Context(), w)
}

func (h *AdminHandler) ProcessBackupSQL(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Gagal parsing form", http.StatusBadRequest)
		return
	}

	dataType := r.FormValue("data_type")
	if dataType == "" {
		dataType = "Semua Data (SQL)"
	}

	_, err := h.DashboardRepo.GetSQLDump()
	if err != nil {
		_ = h.DashboardRepo.LogBackup(dataType, "Failed")
		http.Error(w, "Gagal memproses backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.DashboardRepo.LogBackup(dataType, "Success")
	if err != nil {
		fmt.Printf("[DEBUG] Gagal log riwayat backup: %v\n", err)
	}

	http.Redirect(w, r, "/admin/backup", http.StatusSeeOther)
}

func (h *AdminHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	dump, err := h.DashboardRepo.GetSQLDump()
	if err != nil {
		http.Error(w, "Gagal mengambil data backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fileName := fmt.Sprintf("backup_hostelmart_%s.sql", time.Now().Format("20060102_150405"))

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Length", strconv.Itoa(len(dump)))

	w.Write([]byte(dump))
}