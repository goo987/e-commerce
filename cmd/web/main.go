package main

import (
	"e-commerce/internal/db"
	"e-commerce/internal/handler"
	"e-commerce/internal/repository"
	"e-commerce/internal/router"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	PORT := ":8080"

	// 1. Inisialisasi Database
	database, err := db.Open()
	if err != nil {
		log.Fatal("Gagal membuka database:", err)
	}
	defer database.Close()

	// 2. Inisialisasi Repository
	userRepo := &repository.UserRepository{DB: database}
	productRepo := &repository.ProductRepository{DB: database}
	dashboardRepo := &repository.DashboardRepository{DB: database}
	orderRepo := &repository.OrderRepository{DB: database}
	reviewRepo := &repository.ReviewRepository{DB: database}

	userRepo.EnsureAdminExists()

	// BACKGROUND WORKER: Pembatalan Pesanan Expired (> 24 Jam)
	go func() {
		// Variabel kontrol untuk mencegah proses bertumpuk
		var isRunning bool

		// Pengecekan pertama kali saat aplikasi baru dinyalakan (Startup Check)
		fmt.Println("[SYSTEM] Startup Check: Mengecek pesanan kadaluwarsa...")
		if err := orderRepo.CancelExpiredOrders(); err != nil {
			fmt.Printf("[SYSTEM ERROR] Gagal memproses order expired saat startup: %v\n", err)
		}

		// Interval pengecekan setiap 1 menit sekali
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if isRunning {
				continue
			}

			isRunning = true
			err := orderRepo.CancelExpiredOrders()
			if err != nil {
				fmt.Printf("[SYSTEM ERROR] Gagal memproses order expired: %v\n", err)
			}
			isRunning = false
		}
	}()

	// 3. Inisialisasi Handler
	authH := &handler.AuthHandler{
		UserRepo: userRepo,
	}

	publicH := &handler.PublicHandler{
		ProductRepo: productRepo,
		UserRepo:    userRepo,
		OrderRepo:   orderRepo,
		ReviewRepo:  reviewRepo,
	}

	adminH := &handler.AdminHandler{
		UserRepo:      userRepo,
		ProductRepo:   productRepo,
		DashboardRepo: dashboardRepo,
		OrderRepo:     orderRepo,
	}

	petugasH := &handler.PetugasHandler{
		DashboardRepo: dashboardRepo,
		ProductRepo:   productRepo,
		UserRepo:      userRepo,
		OrderRepo:     orderRepo,
	}

	staticH := handler.NewStaticHandler()

	// 4. Setup Router
	r := router.New(authH, publicH, adminH, petugasH, staticH, dashboardRepo)

	// 5. Jalankan Server
	fmt.Println("Server berjalan di http://localhost" + PORT)

	server := &http.Server{
		Addr:         PORT,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}