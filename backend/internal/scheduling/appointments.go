package scheduling

import (
	"context"
	"time"
)

const (
	StatusConfirmed = "CONFIRMED"
	StatusCancelled = "CANCELLED"
	StatusCompleted = "COMPLETED"
)

type CreateAppointmentRequest struct {
	CustomerID    string
	VehicleID     string
	DealershipID  string
	ServiceTypeID string
	StartTime     time.Time
}

type Appointment struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customerId"`
	VehicleID     string    `json:"vehicleId"`
	DealershipID  string    `json:"dealershipId"`
	ServiceTypeID string    `json:"serviceTypeId"`
	TechnicianID  string    `json:"technicianId"`
	ServiceBayID  string    `json:"serviceBayId"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	Status        string    `json:"status"`
}

type AppointmentListItem struct {
	ID              string    `json:"id"`
	CustomerName    string    `json:"customerName"`
	VehicleVIN      string    `json:"vehicleVin"`
	VehicleLabel    string    `json:"vehicleLabel"`
	DealershipName  string    `json:"dealershipName"`
	ServiceTypeName string    `json:"serviceTypeName"`
	TechnicianName  string    `json:"technicianName"`
	ServiceBayName  string    `json:"serviceBayName"`
	StartTime       time.Time `json:"startTime"`
	EndTime         time.Time `json:"endTime"`
	Status          string    `json:"status"`
}

type AppointmentFilters struct {
	DealershipID *string
	Status       *string
}

type AppointmentDetail struct {
	ID          string           `json:"id"`
	Status      string           `json:"status"`
	StartTime   time.Time        `json:"startTime"`
	EndTime     time.Time        `json:"endTime"`
	Customer    AppointmentRef   `json:"customer"`
	Vehicle     VehicleDetailRef `json:"vehicle"`
	Dealership  AppointmentRef   `json:"dealership"`
	ServiceType AppointmentRef   `json:"serviceType"`
	Technician  AppointmentRef   `json:"technician"`
	ServiceBay  AppointmentRef   `json:"serviceBay"`
}

type AppointmentRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VehicleDetailRef struct {
	ID    string `json:"id"`
	VIN   string `json:"vin"`
	Make  string `json:"make"`
	Model string `json:"model"`
	Year  int    `json:"year"`
}

type AppointmentRepository interface {
	CreateAppointment(ctx context.Context, request CreateAppointmentRequest) (Appointment, error)
	ListAppointments(ctx context.Context, filters AppointmentFilters) ([]AppointmentListItem, error)
	GetAppointmentDetail(ctx context.Context, appointmentID string) (AppointmentDetail, error)
	CancelAppointment(ctx context.Context, appointmentID string) (Appointment, error)
}

type AppointmentService struct {
	repo AppointmentRepository
}

func NewAppointmentService(repo AppointmentRepository) *AppointmentService {
	return &AppointmentService{repo: repo}
}

func (s *AppointmentService) Create(ctx context.Context, request CreateAppointmentRequest) (Appointment, error) {
	return s.repo.CreateAppointment(ctx, request)
}

func (s *AppointmentService) List(ctx context.Context, filters AppointmentFilters) ([]AppointmentListItem, error) {
	return s.repo.ListAppointments(ctx, filters)
}

func (s *AppointmentService) GetDetail(ctx context.Context, appointmentID string) (AppointmentDetail, error) {
	return s.repo.GetAppointmentDetail(ctx, appointmentID)
}

func (s *AppointmentService) Cancel(ctx context.Context, appointmentID string) (Appointment, error) {
	return s.repo.CancelAppointment(ctx, appointmentID)
}

func IsValidAppointmentStatus(status string) bool {
	switch status {
	case StatusConfirmed, StatusCancelled, StatusCompleted:
		return true
	default:
		return false
	}
}
