package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/izz-linux/budget-mgmt/backend/internal/handlers"
)

func New(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Handlers
	billH := handlers.NewBillHandler(db)
	incomeH := handlers.NewIncomeHandler(db)
	periodH := handlers.NewPeriodHandler(db)
	assignH := handlers.NewAssignmentHandler(db)
	gridH := handlers.NewGridHandler(db)
	importH := handlers.NewImportHandler(db)
	optimizerH := handlers.NewOptimizerHandler(db)
	dashboardH := handlers.NewDashboardHandler(db)

	r.Route("/api/v1", func(r chi.Router) {
		// Bills
		r.Get("/bills", billH.List)
		r.Post("/bills", billH.Create)
		r.Get("/bills/{id}", billH.Get)
		r.Put("/bills/{id}", billH.Update)
		r.Delete("/bills/{id}", billH.Delete)
		r.Patch("/bills/reorder", billH.Reorder)

		// Income sources
		r.Get("/income-sources", incomeH.List)
		r.Post("/income-sources", incomeH.Create)
		r.Get("/income-sources/{id}", incomeH.Get)
		r.Put("/income-sources/{id}", incomeH.Update)
		r.Delete("/income-sources/{id}", incomeH.Delete)

		// Pay periods
		r.Get("/pay-periods", periodH.List)
		r.Post("/pay-periods/generate", periodH.Generate)
		r.Put("/pay-periods/{id}", periodH.Update)

		// Bill assignments
		r.Get("/assignments", assignH.List)
		r.Post("/assignments", assignH.Create)
		r.Put("/assignments/{id}", assignH.Update)
		r.Patch("/assignments/{id}/status", assignH.UpdateStatus)
		r.Delete("/assignments/{id}", assignH.Delete)

		// Budget grid (composite view)
		r.Get("/budget-grid", gridH.GetGrid)

		// Import
		r.Post("/import/xlsx", importH.Upload)
		r.Post("/import/xlsx/confirm", importH.Confirm)
		r.Get("/import/history", importH.History)

		// Optimizer
		r.Post("/optimizer/suggest", optimizerH.Suggest)
		r.Get("/optimizer/surplus", optimizerH.Surplus)

		// Dashboard
		r.Get("/dashboard/summary", dashboardH.Summary)
	})

	return r
}
