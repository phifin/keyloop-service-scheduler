CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    vin TEXT NOT NULL UNIQUE,
    make TEXT NOT NULL,
    model TEXT NOT NULL,
    year INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT vehicles_year_check CHECK (year > 1900)
);

CREATE TABLE dealerships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE service_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    duration_minutes INT NOT NULL,
    required_skill_code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT service_types_duration_minutes_check CHECK (duration_minutes > 0)
);

CREATE TABLE technicians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dealership_id UUID NOT NULL REFERENCES dealerships(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE technician_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    technician_id UUID NOT NULL REFERENCES technicians(id),
    skill_code TEXT NOT NULL,
    UNIQUE (technician_id, skill_code)
);

CREATE TABLE service_bays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dealership_id UUID NOT NULL REFERENCES dealerships(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id),
    dealership_id UUID NOT NULL REFERENCES dealerships(id),
    service_type_id UUID NOT NULL REFERENCES service_types(id),
    technician_id UUID NOT NULL REFERENCES technicians(id),
    service_bay_id UUID NOT NULL REFERENCES service_bays(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'CONFIRMED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT appointments_time_check CHECK (end_time > start_time),
    CONSTRAINT appointments_status_check CHECK (status IN ('CONFIRMED', 'CANCELLED', 'COMPLETED'))
);

CREATE INDEX idx_vehicles_customer_id
    ON vehicles(customer_id);

CREATE INDEX idx_technicians_dealership_id
    ON technicians(dealership_id);

CREATE INDEX idx_technician_skills_technician_id_skill_code
    ON technician_skills(technician_id, skill_code);

CREATE INDEX idx_service_bays_dealership_id
    ON service_bays(dealership_id);

CREATE INDEX idx_appointments_dealership_time
    ON appointments(dealership_id, start_time, end_time);

CREATE INDEX idx_appointments_technician_time
    ON appointments(technician_id, start_time, end_time);

CREATE INDEX idx_appointments_service_bay_time
    ON appointments(service_bay_id, start_time, end_time);

CREATE INDEX idx_appointments_status
    ON appointments(status);

CREATE TRIGGER customers_set_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER vehicles_set_updated_at
    BEFORE UPDATE ON vehicles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER dealerships_set_updated_at
    BEFORE UPDATE ON dealerships
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER service_types_set_updated_at
    BEFORE UPDATE ON service_types
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER technicians_set_updated_at
    BEFORE UPDATE ON technicians
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER service_bays_set_updated_at
    BEFORE UPDATE ON service_bays
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER appointments_set_updated_at
    BEFORE UPDATE ON appointments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
