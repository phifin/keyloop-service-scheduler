import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ApiError, cancelAppointment, getAppointment } from '../api/client';
import Alert from '../components/Alert';
import Button from '../components/Button';
import Card from '../components/Card';
import LoadingState from '../components/LoadingState';
import StatusBadge from '../components/StatusBadge';
import type { AppointmentDetail } from '../types/api';

export default function AppointmentDetailPage() {
  const { appointmentId } = useParams();
  const navigate = useNavigate();
  const [appointment, setAppointment] = useState<AppointmentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [canceling, setCanceling] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function loadAppointment() {
    if (!appointmentId) {
      setError('Appointment not found.');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await getAppointment(appointmentId);
      setAppointment(data);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('Appointment not found.');
      } else {
        setError(readableError(err, 'Unable to load appointment.'));
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAppointment();
  }, [appointmentId]);

  async function handleCancel() {
    if (!appointmentId) return;

    try {
      setCanceling(true);
      setError(null);
      await cancelAppointment(appointmentId);
      await loadAppointment();
    } catch (err) {
      setError(readableError(err, 'Unable to cancel appointment.'));
    } finally {
      setCanceling(false);
    }
  }

  if (loading) {
    return <LoadingState label="Loading appointment..." />;
  }

  if (error && !appointment) {
    return (
      <div className="space-y-4">
        <Alert tone="error">{error}</Alert>
        <Button type="button" variant="secondary" onClick={() => navigate('/appointments')}>
          Back to Appointments
        </Button>
      </div>
    );
  }

  if (!appointment) return null;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-slate-950">Appointment Detail</h2>
        </div>
        <div className="flex gap-2">
          <Link className="inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50" to="/appointments">
            Back
          </Link>
          {appointment.status === 'CONFIRMED' && (
            <Button type="button" variant="danger" onClick={handleCancel} disabled={canceling}>
              {canceling ? 'Canceling...' : 'Cancel'}
            </Button>
          )}
        </div>
      </div>

      {error && <Alert tone="error">{error}</Alert>}

      <Card
        title={appointment.serviceType.name}
        actions={<StatusBadge status={appointment.status} />}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Detail label="Customer" value={appointment.customer.name} />
          <Detail
            label="Vehicle"
            value={`${appointment.vehicle.make} ${appointment.vehicle.model}`}
            helper={`${appointment.vehicle.year} - ${appointment.vehicle.vin}`}
          />
          <Detail label="Dealership" value={appointment.dealership.name} />
          <Detail label="Technician" value={appointment.technician.name} />
          <Detail label="Service Bay" value={appointment.serviceBay.name} />
          <Detail label="Start" value={formatDateTime(appointment.startTime)} helper={formatDateTime(appointment.endTime)} />
        </div>
      </Card>
    </div>
  );
}

function Detail({ label, value, helper }: { label: string; value: string; helper?: string }) {
  return (
    <div className="rounded-md bg-slate-50 p-4">
      <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</div>
      <div className="mt-1 font-medium text-slate-950">{value}</div>
      {helper && <div className="mt-1 text-sm text-slate-600">{helper}</div>}
    </div>
  );
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
