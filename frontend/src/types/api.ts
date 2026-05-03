export type AppointmentStatus = 'CONFIRMED' | 'CANCELLED' | 'COMPLETED';

export interface Dealership {
  id: string;
  name: string;
  address: string;
  timezone: string;
}

export interface Customer {
  id: string;
  name: string;
  email: string;
  phone: string | null;
}

export interface Vehicle {
  id: string;
  customerId: string;
  vin: string;
  make: string;
  model: string;
  year: number;
}

export interface ServiceType {
  id: string;
  name: string;
  durationMinutes: number;
  requiredSkillCode: string;
}

export interface Technician {
  id: string;
  dealershipId: string;
  name: string;
  skills: string[];
}

export interface ServiceBay {
  id: string;
  dealershipId: string;
  name: string;
}

export interface AvailabilityResource {
  id: string;
  name: string;
}

export interface AvailabilityResult {
  dealershipId: string;
  serviceTypeId: string;
  startTime: string;
  endTime: string;
  available: boolean;
  availableTechnicians: AvailabilityResource[];
  availableServiceBays: AvailabilityResource[];
  reason: string | null;
}

export interface CreateAppointmentRequest {
  customerId: string;
  vehicleId: string;
  dealershipId: string;
  serviceTypeId: string;
  startTime: string;
}

export interface Appointment {
  id: string;
  customerId: string;
  vehicleId: string;
  dealershipId: string;
  serviceTypeId: string;
  technicianId: string;
  serviceBayId: string;
  startTime: string;
  endTime: string;
  status: AppointmentStatus;
}

export interface AppointmentListItem {
  id: string;
  customerName: string;
  vehicleVin: string;
  vehicleLabel: string;
  dealershipName: string;
  serviceTypeName: string;
  technicianName: string;
  serviceBayName: string;
  startTime: string;
  endTime: string;
  status: AppointmentStatus;
}

export interface AppointmentRef {
  id: string;
  name: string;
}

export interface AppointmentVehicle {
  id: string;
  vin: string;
  make: string;
  model: string;
  year: number;
}

export interface AppointmentDetail {
  id: string;
  status: AppointmentStatus;
  startTime: string;
  endTime: string;
  customer: AppointmentRef;
  vehicle: AppointmentVehicle;
  dealership: AppointmentRef;
  serviceType: AppointmentRef;
  technician: AppointmentRef;
  serviceBay: AppointmentRef;
}

export interface ApiErrorBody {
  error: string;
  message: string;
  code?: string;
}
