package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"keyloop-service-scheduler/internal/http/handlers"
	"keyloop-service-scheduler/internal/repository"
	"keyloop-service-scheduler/internal/scheduling"
)

func NewRouter(referenceRepo repository.ReferenceRepository, availabilityService *scheduling.AvailabilityService, appointmentService handlers.AppointmentService, logger *slog.Logger, middlewares ...func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()

	for _, middleware := range middlewares {
		router.Use(middleware)
	}

	referenceHandler := handlers.NewReferenceHandler(referenceRepo)
	availabilityHandler := handlers.NewAvailabilityHandler(availabilityService, logger)
	appointmentHandler := handlers.NewAppointmentHandler(appointmentService, logger)

	router.Get("/health", handlers.Health)
	router.Get("/availability", availabilityHandler.Check)
	router.Get("/appointments", appointmentHandler.List)
	router.Post("/appointments", appointmentHandler.Create)
	router.Get("/appointments/{appointmentId}", appointmentHandler.GetDetail)
	router.Patch("/appointments/{appointmentId}/cancel", appointmentHandler.Cancel)
	router.Get("/dealerships", referenceHandler.ListDealerships)
	router.Get("/customers", referenceHandler.ListCustomers)
	router.Get("/customers/{customerId}/vehicles", referenceHandler.ListCustomerVehicles)
	router.Get("/service-types", referenceHandler.ListServiceTypes)
	router.Get("/technicians", referenceHandler.ListTechnicians)
	router.Get("/service-bays", referenceHandler.ListServiceBays)

	return router
}
