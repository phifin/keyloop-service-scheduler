package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"keyloop-service-scheduler/internal/repository"
)

type ReferenceHandler struct {
	repo repository.ReferenceRepository
}

func NewReferenceHandler(repo repository.ReferenceRepository) *ReferenceHandler {
	return &ReferenceHandler{repo: repo}
}

func (h *ReferenceHandler) ListDealerships(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "database is not configured")
		return
	}

	dealerships, err := h.repo.ListDealerships(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load dealerships")
		return
	}

	writeJSON(w, http.StatusOK, dealerships)
}

func (h *ReferenceHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "database is not configured")
		return
	}

	customers, err := h.repo.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load customers")
		return
	}

	writeJSON(w, http.StatusOK, customers)
}

func (h *ReferenceHandler) ListCustomerVehicles(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "database is not configured")
		return
	}

	customerID := chi.URLParam(r, "customerId")
	if _, err := uuid.Parse(customerID); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "customerId must be a valid UUID")
		return
	}

	vehicles, err := h.repo.ListCustomerVehicles(r.Context(), customerID)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "customer not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load vehicles")
		return
	}

	writeJSON(w, http.StatusOK, vehicles)
}

func (h *ReferenceHandler) ListServiceTypes(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "database is not configured")
		return
	}

	serviceTypes, err := h.repo.ListServiceTypes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load service types")
		return
	}

	writeJSON(w, http.StatusOK, serviceTypes)
}

func (h *ReferenceHandler) ListTechnicians(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "database is not configured")
		return
	}

	dealershipID, ok := requiredUUIDQuery(w, r, "dealershipId")
	if !ok {
		return
	}

	technicians, err := h.repo.ListTechnicians(r.Context(), dealershipID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load technicians")
		return
	}

	writeJSON(w, http.StatusOK, technicians)
}

func (h *ReferenceHandler) ListServiceBays(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "database is not configured")
		return
	}

	dealershipID, ok := requiredUUIDQuery(w, r, "dealershipId")
	if !ok {
		return
	}

	serviceBays, err := h.repo.ListServiceBays(r.Context(), dealershipID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load service bays")
		return
	}

	writeJSON(w, http.StatusOK, serviceBays)
}

func requiredUUIDQuery(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		writeError(w, http.StatusBadRequest, "bad_request", name+" is required")
		return "", false
	}

	if _, err := uuid.Parse(value); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", name+" must be a valid UUID")
		return "", false
	}

	return value, true
}
