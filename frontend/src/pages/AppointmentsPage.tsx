import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { ApiError, cancelAppointment, listAppointments } from '../api/client';
import Alert from '../components/Alert';
import Button from '../components/Button';
import Card from '../components/Card';
import EmptyState from '../components/EmptyState';
import Field, { inputClass } from '../components/Field';
import LoadingState from '../components/LoadingState';
import StatusBadge from '../components/StatusBadge';
import type { AppointmentListItem, AppointmentStatus } from '../types/api';

export default function AppointmentsPage() {
  const [appointments, setAppointments] = useState<AppointmentListItem[]>([]);
  const [status, setStatus] = useState<AppointmentStatus | ''>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [cancelingId, setCancelingId] = useState<string | null>(null);

  async function loadAppointments() {
    try {
      setLoading(true);
      setError(null);
      const data = await listAppointments({ status });
      setAppointments(data);
    } catch (err) {
      setError(readableError(err, 'Unable to load appointments.'));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAppointments();
  }, [status]);

  async function handleCancel(appointmentId: string) {
    try {
      setCancelingId(appointmentId);
      setError(null);
      await cancelAppointment(appointmentId);
      await loadAppointments();
    } catch (err) {
      setError(readableError(err, 'Unable to cancel appointment.'));
    } finally {
      setCancelingId(null);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-slate-950">Appointments</h2>
        </div>
        <div className="w-full sm:w-56">
          <Field label="Status">
            <select className={inputClass} value={status} onChange={(event) => setStatus(event.target.value as AppointmentStatus | '')}>
              <option value="">All statuses</option>
              <option value="CONFIRMED">Confirmed</option>
              <option value="CANCELLED">Cancelled</option>
              <option value="COMPLETED">Completed</option>
            </select>
          </Field>
        </div>
      </div>

      {error && <Alert tone="error">{error}</Alert>}

      {loading ? (
        <LoadingState label="Loading appointments..." />
      ) : (
        <Card>
          {appointments.length === 0 ? (
            <EmptyState title="No appointments found." />
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-200 text-sm">
                <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
                  <tr>
                    <th className="px-3 py-3">Customer</th>
                    <th className="px-3 py-3">Vehicle</th>
                    <th className="px-3 py-3">Service</th>
                    <th className="px-3 py-3">Resources</th>
                    <th className="px-3 py-3">Time</th>
                    <th className="px-3 py-3">Status</th>
                    <th className="px-3 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 bg-white">
                  {appointments.map((appointment) => (
                    <tr key={appointment.id}>
                      <td className="px-3 py-3 font-medium text-slate-950">{appointment.customerName}</td>
                      <td className="px-3 py-3">
                        <div>{appointment.vehicleLabel}</div>
                        <div className="text-xs text-slate-500">{appointment.vehicleVin}</div>
                      </td>
                      <td className="px-3 py-3">{appointment.serviceTypeName}</td>
                      <td className="px-3 py-3">
                        <div>{appointment.technicianName}</div>
                        <div className="text-xs text-slate-500">{appointment.serviceBayName}</div>
                      </td>
                      <td className="px-3 py-3">
                        <div>{formatDateTime(appointment.startTime)}</div>
                        <div className="text-xs text-slate-500">{formatDateTime(appointment.endTime)}</div>
                      </td>
                      <td className="px-3 py-3">
                        <StatusBadge status={appointment.status} />
                      </td>
                      <td className="space-x-2 whitespace-nowrap px-3 py-3 text-right">
                        <Link className="text-sm font-semibold text-sky-700 hover:text-sky-900" to={`/appointments/${appointment.id}`}>
                          View
                        </Link>
                        {appointment.status === 'CONFIRMED' && (
                          <Button
                            type="button"
                            variant="danger"
                            className="px-3 py-1.5"
                            onClick={() => handleCancel(appointment.id)}
                            disabled={cancelingId === appointment.id}
                          >
                            {cancelingId === appointment.id ? 'Canceling...' : 'Cancel'}
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      )}
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
