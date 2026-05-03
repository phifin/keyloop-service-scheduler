package scheduling

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOverlaps(t *testing.T) {
	parse := func(value string) time.Time {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}

	tests := []struct {
		name          string
		existingStart time.Time
		existingEnd   time.Time
		newStart      time.Time
		newEnd        time.Time
		expected      bool
	}{
		{
			name:          "partial overlap",
			existingStart: parse("2026-05-04T09:00:00Z"),
			existingEnd:   parse("2026-05-04T10:00:00Z"),
			newStart:      parse("2026-05-04T09:30:00Z"),
			newEnd:        parse("2026-05-04T10:30:00Z"),
			expected:      true,
		},
		{
			name:          "adjacent intervals do not overlap",
			existingStart: parse("2026-05-04T09:00:00Z"),
			existingEnd:   parse("2026-05-04T10:00:00Z"),
			newStart:      parse("2026-05-04T10:00:00Z"),
			newEnd:        parse("2026-05-04T11:00:00Z"),
			expected:      false,
		},
		{
			name:          "adjacent interval before does not overlap",
			existingStart: parse("2026-05-04T09:00:00Z"),
			existingEnd:   parse("2026-05-04T10:00:00Z"),
			newStart:      parse("2026-05-04T08:00:00Z"),
			newEnd:        parse("2026-05-04T09:00:00Z"),
			expected:      false,
		},
		{
			name:          "existing contains new interval",
			existingStart: parse("2026-05-04T09:00:00Z"),
			existingEnd:   parse("2026-05-04T12:00:00Z"),
			newStart:      parse("2026-05-04T10:00:00Z"),
			newEnd:        parse("2026-05-04T11:00:00Z"),
			expected:      true,
		},
		{
			name:          "new contains existing interval",
			existingStart: parse("2026-05-04T10:00:00Z"),
			existingEnd:   parse("2026-05-04T11:00:00Z"),
			newStart:      parse("2026-05-04T09:00:00Z"),
			newEnd:        parse("2026-05-04T12:00:00Z"),
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Overlaps(tt.existingStart, tt.existingEnd, tt.newStart, tt.newEnd)
			if actual != tt.expected {
				t.Fatalf("expected %t, got %t", tt.expected, actual)
			}
		})
	}
}

func TestAvailabilityServiceUnknownDealership(t *testing.T) {
	service := NewAvailabilityService(fakeAvailabilityRepository{dealershipErr: ErrNotFound})

	_, err := service.Check(context.Background(), validAvailabilityRequest(t))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAvailabilityServiceUnknownServiceType(t *testing.T) {
	service := NewAvailabilityService(fakeAvailabilityRepository{serviceTypeErr: ErrNotFound})

	_, err := service.Check(context.Background(), validAvailabilityRequest(t))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAvailabilityServiceExcludesBusyResources(t *testing.T) {
	service := NewAvailabilityService(fakeAvailabilityRepository{
		busyTechnicianIDs: []string{"tech-1"},
		busyServiceBayIDs: []string{"bay-1"},
	})

	result, err := service.Check(context.Background(), validAvailabilityRequest(t))
	if err != nil {
		t.Fatal(err)
	}

	if !result.Available {
		t.Fatal("expected availability with remaining technician and service bay")
	}
	if len(result.AvailableTechnicians) != 1 || result.AvailableTechnicians[0].ID != "tech-2" {
		t.Fatalf("unexpected available technicians: %+v", result.AvailableTechnicians)
	}
	if len(result.AvailableServiceBays) != 1 || result.AvailableServiceBays[0].ID != "bay-2" {
		t.Fatalf("unexpected available service bays: %+v", result.AvailableServiceBays)
	}
}

func validAvailabilityRequest(t *testing.T) AvailabilityRequest {
	t.Helper()

	startTime, err := time.Parse(time.RFC3339, "2026-05-04T09:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	return AvailabilityRequest{
		DealershipID:  "44444444-4444-4444-4444-444444444444",
		ServiceTypeID: "55555555-5555-5555-5555-555555555551",
		StartTime:     startTime,
	}
}

type fakeAvailabilityRepository struct {
	dealershipErr     error
	serviceTypeErr    error
	busyTechnicianIDs []string
	busyServiceBayIDs []string
}

func (r fakeAvailabilityRepository) VerifyDealershipExists(ctx context.Context, dealershipID string) error {
	return r.dealershipErr
}

func (r fakeAvailabilityRepository) GetServiceType(ctx context.Context, serviceTypeID string) (ServiceType, error) {
	if r.serviceTypeErr != nil {
		return ServiceType{}, r.serviceTypeErr
	}

	return ServiceType{
		ID:                serviceTypeID,
		DurationMinutes:   30,
		RequiredSkillCode: "OIL_CHANGE",
	}, nil
}

func (r fakeAvailabilityRepository) ListQualifiedTechnicians(ctx context.Context, dealershipID, requiredSkillCode string) ([]Resource, error) {
	return []Resource{
		{ID: "tech-1", Name: "Alex Morgan"},
		{ID: "tech-2", Name: "Taylor Smith"},
	}, nil
}

func (r fakeAvailabilityRepository) ListBusyTechnicianIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error) {
	return r.busyTechnicianIDs, nil
}

func (r fakeAvailabilityRepository) ListAvailableServiceBays(ctx context.Context, dealershipID string) ([]Resource, error) {
	return []Resource{
		{ID: "bay-1", Name: "Bay 1"},
		{ID: "bay-2", Name: "Bay 2"},
	}, nil
}

func (r fakeAvailabilityRepository) ListBusyServiceBayIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error) {
	return r.busyServiceBayIDs, nil
}
