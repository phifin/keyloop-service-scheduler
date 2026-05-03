import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  ApiError,
  checkAvailability,
  createAppointment,
  listCustomerVehicles,
  listCustomers,
  listDealerships,
  listServiceTypes,
} from '../api/client';
import Alert from '../components/Alert';
import Button from '../components/Button';
import Card from '../components/Card';
import Field, { inputClass } from '../components/Field';
import LoadingState from '../components/LoadingState';
import type { AvailabilityResult, Customer, Dealership, ServiceType, Vehicle } from '../types/api';

export default function BookingPage() {
  const navigate = useNavigate();
  const [dealerships, setDealerships] = useState<Dealership[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([]);
  const [dealershipId, setDealershipId] = useState('');
  const [customerId, setCustomerId] = useState('');
  const [vehicleId, setVehicleId] = useState('');
  const [serviceTypeId, setServiceTypeId] = useState('');
  const [startTime, setStartTime] = useState('');
  const [availability, setAvailability] = useState<AvailabilityResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [vehiclesLoading, setVehiclesLoading] = useState(false);
  const [checking, setChecking] = useState(false);
  const [booking, setBooking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    async function loadReferenceData() {
      try {
        setLoading(true);
        const [dealershipData, customerData, serviceTypeData] = await Promise.all([
          listDealerships(),
          listCustomers(),
          listServiceTypes(),
        ]);
        if (!active) return;
        setDealerships(dealershipData);
        setCustomers(customerData);
        setServiceTypes(serviceTypeData);
        setDealershipId(dealershipData[0]?.id ?? '');
      } catch (err) {
        if (active) setError(readableError(err, 'Unable to load booking data.'));
      } finally {
        if (active) setLoading(false);
      }
    }

    loadReferenceData();
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    setAvailability(null);
    setSuccess(null);
  }, [dealershipId, vehicleId, serviceTypeId, startTime]);

  useEffect(() => {
    let active = true;
    setVehicleId('');
    setVehicles([]);
    setAvailability(null);
    setSuccess(null);
    setVehiclesLoading(false);

    if (!customerId) {
      return () => {
        active = false;
      };
    }

    async function loadVehicles() {
      try {
        setVehiclesLoading(true);
        const data = await listCustomerVehicles(customerId);
        if (!active) return;
        setVehicles(data);
        setVehicleId(data[0]?.id ?? '');
      } catch (err) {
        if (active) setError(readableError(err, 'Unable to load vehicles.'));
      } finally {
        if (active) setVehiclesLoading(false);
      }
    }

    loadVehicles();
    return () => {
      active = false;
    };
  }, [customerId]);

  const selectedServiceType = useMemo(
    () => serviceTypes.find((serviceType) => serviceType.id === serviceTypeId),
    [serviceTypeId, serviceTypes],
  );
  const isoStartTime = toIsoStartTime(startTime);
  const bookingFormComplete = Boolean(customerId && vehicleId && dealershipId && serviceTypeId && isoStartTime);
  const canConfirmBooking = Boolean(availability?.available && bookingFormComplete && !vehiclesLoading && !booking);

  async function handleCheckAvailability() {
    setError(null);
    setSuccess(null);
    setAvailability(null);

    if (!dealershipId || !serviceTypeId || !isoStartTime) {
      setError('Select a dealership, service type, and start date/time.');
      return;
    }

    try {
      setChecking(true);
      const result = await checkAvailability({
        dealershipId,
        serviceTypeId,
        startTime: isoStartTime,
      });
      setAvailability(result);
    } catch (err) {
      setAvailability(null);
      setError(readableError(err, 'Unable to check availability.'));
    } finally {
      setChecking(false);
    }
  }

  async function handleConfirmBooking() {
    setError(null);
    setSuccess(null);

    if (!availability?.available || !bookingFormComplete || !isoStartTime) {
      setError('Check availability before confirming the booking.');
      return;
    }

    try {
      setBooking(true);
      const appointment = await createAppointment({
        customerId,
        vehicleId,
        dealershipId,
        serviceTypeId,
        startTime: isoStartTime,
      });
      setSuccess('Appointment confirmed.');
      navigate(`/appointments/${appointment.id}`);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setAvailability(null);
        setError('This slot is no longer available. Please check availability again.');
      } else {
        setError(readableError(err, 'Unable to create appointment.'));
      }
    } finally {
      setBooking(false);
    }
  }

  if (loading) {
    return <LoadingState label="Loading booking data..." />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold text-slate-950">Book Appointment</h2>
      </div>

      {error && <Alert tone="error">{error}</Alert>}
      {success && <Alert tone="success">{success}</Alert>}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <Card title="Appointment Details">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Dealership">
              <select className={inputClass} value={dealershipId} onChange={(event) => setDealershipId(event.target.value)}>
                <option value="">Select dealership</option>
                {dealerships.map((dealership) => (
                  <option key={dealership.id} value={dealership.id}>
                    {dealership.name}
                  </option>
                ))}
              </select>
            </Field>

            <Field label="Customer">
              <select className={inputClass} value={customerId} onChange={(event) => setCustomerId(event.target.value)}>
                <option value="">Select customer</option>
                {customers.map((customer) => (
                  <option key={customer.id} value={customer.id}>
                    {customer.name}
                  </option>
                ))}
              </select>
            </Field>

            <Field label="Vehicle">
              <select
                className={inputClass}
                value={vehicleId}
                onChange={(event) => setVehicleId(event.target.value)}
                disabled={!customerId || vehiclesLoading}
              >
                <option value="">{vehiclesLoading ? 'Loading vehicles' : 'Select vehicle'}</option>
                {vehicles.map((vehicle) => (
                  <option key={vehicle.id} value={vehicle.id}>
                    {vehicle.make} {vehicle.model} - {vehicle.vin}
                  </option>
                ))}
              </select>
            </Field>

            <Field label="Service Type">
              <select className={inputClass} value={serviceTypeId} onChange={(event) => setServiceTypeId(event.target.value)}>
                <option value="">Select service</option>
                {serviceTypes.map((serviceType) => (
                  <option key={serviceType.id} value={serviceType.id}>
                    {serviceType.name} ({serviceType.durationMinutes} min)
                  </option>
                ))}
              </select>
            </Field>

            <Field label="Start Date/Time">
              <input
                className={inputClass}
                type="datetime-local"
                value={startTime}
                onChange={(event) => setStartTime(event.target.value)}
              />
            </Field>
          </div>

          <div className="mt-5 flex flex-col gap-3 sm:flex-row">
            <Button type="button" onClick={handleCheckAvailability} disabled={checking}>
              {checking ? 'Checking...' : 'Check Availability'}
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={handleConfirmBooking}
              disabled={!canConfirmBooking}
            >
              {booking ? 'Confirming...' : 'Confirm Booking'}
            </Button>
          </div>
        </Card>

        <Card title="Availability">
          {!availability && (
            <div className="space-y-3 text-sm text-slate-600">
              <p>{selectedServiceType ? `${selectedServiceType.name} requires ${selectedServiceType.durationMinutes} minutes.` : 'Select a service type.'}</p>
            </div>
          )}

          {availability && (
            <div className="space-y-4">
              <Alert tone={availability.available ? 'success' : 'error'}>
                {availability.available ? 'Available' : availability.reason}
              </Alert>

              <dl className="grid grid-cols-2 gap-3 text-sm">
                <div className="rounded-md bg-slate-50 p-3">
                  <dt className="text-slate-500">Technicians</dt>
                  <dd className="text-lg font-semibold">{availability.availableTechnicians.length}</dd>
                </div>
                <div className="rounded-md bg-slate-50 p-3">
                  <dt className="text-slate-500">Service Bays</dt>
                  <dd className="text-lg font-semibold">{availability.availableServiceBays.length}</dd>
                </div>
                <div className="col-span-2 rounded-md bg-slate-50 p-3">
                  <dt className="text-slate-500">End Time</dt>
                  <dd className="font-medium">{formatDateTime(availability.endTime)}</dd>
                </div>
              </dl>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}

function toIsoStartTime(value: string): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toISOString();
}

function readableError(error: unknown, fallback: string) {
  if (error instanceof ApiError) return error.body?.message ?? fallback;
  if (error instanceof Error) return error.message;
  return fallback;
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
