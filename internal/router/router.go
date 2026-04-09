package router

import (
	"e-commerce/internal/handler"
	authMiddleware "e-commerce/internal/middleware"
	"e-commerce/internal/repository"
	"e-commerce/views"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(
	authH *handler.AuthHandler,
	publicH *handler.PublicHandler,
	adminH *handler.AdminHandler,
	petugasH *handler.PetugasHandler,
	staticH *handler.StaticHandler,
	dashRepo *repository.DashboardRepository,
) http.Handler {

	r := chi.NewRouter()

	// GLOBAL MIDDLEWARE
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// STATIC FILE & UPLOADS
	r.Handle("/image/*", http.StripPrefix("/image/", http.FileServer(http.Dir("./image"))))
	r.Handle("/bukti_transfer/*", http.StripPrefix("/bukti_transfer/", http.FileServer(http.Dir("./bukti_transfer"))))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// STATIC PAGES (PUBLIC)
	r.Get("/tentang-kami", staticH.TentangKami)
	r.Get("/cara-belanja", staticH.CaraBelanja)
	r.Get("/pembayaran", staticH.MetodePembayaran)
	r.Get("/pengiriman", staticH.Pengiriman)
	r.Get("/syarat-ketentuan", staticH.SyaratKetentuan)

	// PUBLIC (OPTIONAL AUTH)
	r.Group(func(public chi.Router) {
		public.Use(authMiddleware.OptionalAuth)

		public.Get("/", publicH.Home)
		public.Get("/produk/{id}", publicH.ProductDetail)
	})

	// AUTH ROUTES
	r.Get("/register", authH.ShowRegister)
	r.Post("/register", authH.Register)
	r.Get("/login", authH.ShowLogin)
	r.Post("/login", authH.Login)
	r.Get("/logout", authH.Logout)

	// USER ROUTES (LOGIN REQUIRED)
	r.Group(func(user chi.Router) {
		user.Use(authMiddleware.TrackVisitor(dashRepo))
		user.Use(authMiddleware.RequireRole("user"))

		user.Get("/keranjang", publicH.Cart)
		user.Post("/keranjang/add", publicH.AddToCart)
		user.Get("/keranjang/update", publicH.UpdateCartQty)
		user.Get("/keranjang/hapus/{id}", publicH.RemoveFromCart)
		user.Get("/keranjang/hapus-masal", publicH.BulkRemoveFromCart)

		// Checkout & Order
		user.Get("/checkout", publicH.Checkout)
		user.Post("/proses-checkout", publicH.ProsesCheckout)

		// Riwayat Pesanan & Detail
		user.Get("/riwayat", publicH.OrderHistory)
		user.Get("/riwayat/detail/{id}", publicH.OrderDetail)

		// Pembatalan Pesanan
		user.Post("/order/cancel", publicH.BatalkanPesanan)
		user.Post("/review/create", publicH.SubmitReview)

		// Pembayaran & Konfirmasi
		user.Get("/pembayaran/{id}", publicH.Pembayaran)
		user.Post("/konfirmasi-pembayaran", publicH.KonfirmasiPembayaran)

		// Profil & Akun
		user.Get("/akun", publicH.Akun)
		user.Post("/akun/update", publicH.UpdateAkun)
		user.Post("/akun/change-password", publicH.ChangePassword)
		user.Post("/akun/update-photo", publicH.UpdateFotoProfil)
		user.Post("/akun/delete-photo", publicH.DeleteFotoProfil)
	})

	// ADMIN ROUTES
	r.Route("/admin", func(admin chi.Router) {
		admin.Use(authMiddleware.RequireRole("admin"))
		admin.Get("/dashboard", adminH.AdminDashboard)
		admin.Get("/laporan", adminH.AdminLaporan)
		admin.Get("/laporan/detail/{id}", adminH.DetailPesananLaporan)

		admin.Get("/petugas", adminH.AdminPetugas)
		admin.Post("/petugas/create", adminH.CreatePetugas)
		admin.Post("/petugas/update", adminH.UpdatePetugas)
		admin.Get("/petugas/toggle", adminH.TogglePetugasStatus)
		admin.Get("/petugas/delete", adminH.DeletePetugas)

		admin.Get("/user", adminH.AdminUser)
		admin.Get("/user/toggle", adminH.ToggleUserStatus)

		admin.Get("/produk", adminH.AdminProduk)
		admin.Post("/produk/create", adminH.CreateProduk)
		admin.Post("/produk/update", adminH.UpdateProduk)
		admin.Get("/produk/delete", adminH.DeleteProduk)

		admin.Get("/backup", adminH.AdminBackup)
		admin.Post("/backup/process-sql", adminH.ProcessBackupSQL)
		admin.Get("/backup/download", adminH.DownloadBackup)
	})

	// PETUGAS ROUTES
	r.Route("/petugas", func(p chi.Router) {
		p.Use(authMiddleware.RequireRole("petugas"))
		p.Get("/dashboard", petugasH.PetugasDashboard)
		p.Get("/laporan", petugasH.PetugasLaporan)
		p.Get("/laporan/detail/{id}", petugasH.DetailPesananLaporan)

		p.Get("/pesanan", petugasH.PetugasPesanan)
		p.Post("/order/update-status", petugasH.UpdateOrderStatus)

		p.Post("/order/update-tracking-step", petugasH.UpdateTrackingStep)

		p.Get("/pesanan/detail/{id}", petugasH.DetailPesanan)
		p.Post("/pesanan/selesai/{id}", petugasH.SelesaikanPesanan)
		p.Post("/pesanan/cancel/{id}", petugasH.KonfirmasiPembatalan)

		p.Get("/user", petugasH.PetugasUser)
		p.Get("/user/toggle", petugasH.ToggleUserStatus)

		p.Get("/produk", petugasH.PetugasProduk)
		p.Post("/produk/create", petugasH.CreateProduk)
		p.Post("/produk/update", petugasH.UpdateProduk)
		p.Get("/produk/delete", petugasH.DeleteProduk)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		views.Notfound().Render(r.Context(), w)
	})

	return r
}