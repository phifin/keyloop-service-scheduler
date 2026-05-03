import type {
  ApiErrorBody,
  Appointment,
  AppointmentDetail,
  AppointmentListItem,
  AppointmentStatus,
  AvailabilityResult,
  CreateAppointmentRequest,
  Customer,
  Dealership,
  ServiceBay,
  ServiceType,
  Technician,
  Vehicle,
} from '../types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';

export class ApiError extends Error {
  status: number;
  body: ApiErrorBody | null;

  constructor(status: number, body: ApiErrorBody | null) {
    super(body?.message ?? `Request failed with status ${status}`);
    this.status = status;
    this.body = body;
  }
}

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
    ...init,
  });

  const text = await response.text();
  const data = text ? (JSON.parse(text) as unknown) : null;

  if (!response.ok) {
    throw new ApiError(response.status, isApiErrorBody(data) ? data : null);
  }

  return data as T;
}

function isApiErrorBody(value: unknown): value is ApiErrorBody {
  return (
    typeof value === 'object' &&
    value !== null &&
    'error' in value &&
    'message' in value &&
    typeof (value as ApiErrorBody).error === 'string' &&
    typeof (value as ApiErrorBody).message === 'string'
  );
}

function searchParams(params: Record<string, string | undefined>): string {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value) {
      query.set(key, value);
    }
  });
  const value = query.toString();
  return value ? `?${value}` : '';
}

export function listDealerships() {
  return apiRequest<Dealership[]>('/dealerships');
}

export function listCustomers() {
  return apiRequest<Customer[]>('/customers');
}

export function listCustomerVehicles(customerId: string) {
  return apiRequest<Vehicle[]>(`/customers/${customerId}/vehicles`);
}

export function listServiceTypes() {
  return apiRequest<ServiceType[]>('/service-types');
}

export function listTechnicians(dealershipId: string) {
  return apiRequest<Technician[]>(`/technicians${searchParams({ dealershipId })}`);
}

export function listServiceBays(dealershipId: string) {
  return apiRequest<ServiceBay[]>(`/service-bays${searchParams({ dealershipId })}`);
}

export function checkAvailability(params: {
  dealershipId: string;
  serviceTypeId: string;
  startTime: string;
}) {
  return apiRequest<AvailabilityResult>(`/availability${searchParams(params)}`);
}

export function createAppointment(request: CreateAppointmentRequest) {
  return apiRequest<Appointment>('/appointments', {
    method: 'POST',
    body: JSON.stringify(request),
  });
}

export function listAppointments(filters?: {
  dealershipId?: string;
  status?: AppointmentStatus | '';
}) {
  return apiRequest<AppointmentListItem[]>(
    `/appointments${searchParams({
      dealershipId: filters?.dealershipId,
      status: filters?.status || undefined,
    })}`,
  );
}

export function getAppointment(appointmentId: string) {
  return apiRequest<AppointmentDetail>(`/appointments/${appointmentId}`);
}

export function cancelAppointment(appointmentId: string) {
  return apiRequest<Appointment>(`/appointments/${appointmentId}/cancel`, {
    method: 'PATCH',
  });
}
