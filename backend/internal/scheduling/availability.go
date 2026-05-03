package scheduling

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("resource unavailable")
var ErrVehicleCustomerMismatch = errors.New("vehicle does not belong to customer")
var ErrCompletedAppointmentCannotCancel = errors.New("completed appointment cannot be cancelled")

type ServiceType struct {
	ID                string
	DurationMinutes   int
	RequiredSkillCode string
}

type Resource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AvailabilityRequest struct {
	DealershipID  string
	ServiceTypeID string
	StartTime     time.Time
}

type AvailabilityResult struct {
	DealershipID         string     `json:"dealershipId"`
	ServiceTypeID        string     `json:"serviceTypeId"`
	StartTime            time.Time  `json:"startTime"`
	EndTime              time.Time  `json:"endTime"`
	Available            bool       `json:"available"`
	AvailableTechnicians []Resource `json:"availableTechnicians"`
	AvailableServiceBays []Resource `json:"availableServiceBays"`
	Reason               *string    `json:"reason"`
}

type AvailabilityRepository interface {
	VerifyDealershipExists(ctx context.Context, dealershipID string) error
	GetServiceType(ctx context.Context, serviceTypeID string) (ServiceType, error)
	ListQualifiedTechnicians(ctx context.Context, dealershipID, requiredSkillCode string) ([]Resource, error)
	ListBusyTechnicianIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error)
	ListAvailableServiceBays(ctx context.Context, dealershipID string) ([]Resource, error)
	ListBusyServiceBayIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error)
}

type AvailabilityService struct {
	repo AvailabilityRepository
}

func NewAvailabilityService(repo AvailabilityRepository) *AvailabilityService {
	return &AvailabilityService{repo: repo}
}

// Check returns a point-in-time availability snapshot. Appointment creation must
// re-check availability transactionally before inserting a confirmed booking.
func (s *AvailabilityService) Check(ctx context.Context, request AvailabilityRequest) (AvailabilityResult, error) {
	if s.repo == nil {
		return AvailabilityResult{}, errors.New("availability repository is not configured")
	}

	if err := s.repo.VerifyDealershipExists(ctx, request.DealershipID); err != nil {
		return AvailabilityResult{}, err
	}

	serviceType, err := s.repo.GetServiceType(ctx, request.ServiceTypeID)
	if err != nil {
		return AvailabilityResult{}, err
	}

	endTime := request.StartTime.Add(time.Duration(serviceType.DurationMinutes) * time.Minute)

	qualifiedTechnicians, err := s.repo.ListQualifiedTechnicians(ctx, request.DealershipID, serviceType.RequiredSkillCode)
	if err != nil {
		return AvailabilityResult{}, err
	}

	busyTechnicianIDs, err := s.repo.ListBusyTechnicianIDs(ctx, request.DealershipID, request.StartTime, endTime)
	if err != nil {
		return AvailabilityResult{}, err
	}

	serviceBays, err := s.repo.ListAvailableServiceBays(ctx, request.DealershipID)
	if err != nil {
		return AvailabilityResult{}, err
	}

	busyServiceBayIDs, err := s.repo.ListBusyServiceBayIDs(ctx, request.DealershipID, request.StartTime, endTime)
	if err != nil {
		return AvailabilityResult{}, err
	}

	availableTechnicians := removeBusyResources(qualifiedTechnicians, busyTechnicianIDs)
	availableServiceBays := removeBusyResources(serviceBays, busyServiceBayIDs)

	result := AvailabilityResult{
		DealershipID:         request.DealershipID,
		ServiceTypeID:        request.ServiceTypeID,
		StartTime:            request.StartTime.UTC(),
		EndTime:              endTime.UTC(),
		Available:            len(availableTechnicians) > 0 && len(availableServiceBays) > 0,
		AvailableTechnicians: availableTechnicians,
		AvailableServiceBays: availableServiceBays,
	}

	if !result.Available {
		result.Reason = unavailableReason(len(availableTechnicians), len(availableServiceBays))
	}

	return result, nil
}

func Overlaps(existingStart, existingEnd, newStart, newEnd time.Time) bool {
	return existingStart.Before(newEnd) && existingEnd.After(newStart)
}

func removeBusyResources(resources []Resource, busyIDs []string) []Resource {
	busy := make(map[string]struct{}, len(busyIDs))
	for _, id := range busyIDs {
		busy[id] = struct{}{}
	}

	available := make([]Resource, 0, len(resources))
	for _, resource := range resources {
		if _, ok := busy[resource.ID]; !ok {
			available = append(available, resource)
		}
	}

	return available
}

func unavailableReason(technicianCount, serviceBayCount int) *string {
	var reason string
	switch {
	case technicianCount == 0 && serviceBayCount == 0:
		reason = "No qualified technician or service bay is available for the requested time range."
	case technicianCount == 0:
		reason = "No qualified technician is available for the requested time range."
	default:
		reason = "No service bay is available for the requested time range."
	}

	return &reason
}
