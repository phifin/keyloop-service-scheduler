INSERT INTO customers (id, name, email, phone)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'John Smith', 'john.smith@example.com', '+1-555-0101'),
    ('22222222-2222-2222-2222-222222222222', 'Maria Garcia', 'maria.garcia@example.com', '+1-555-0102'),
    ('33333333-3333-3333-3333-333333333333', 'David Nguyen', 'david.nguyen@example.com', '+1-555-0103')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone;

INSERT INTO vehicles (id, customer_id, vin, make, model, year)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', '11111111-1111-1111-1111-111111111111', 'KLSSVIN000000001', 'Toyota', 'Corolla', 2021),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2', '22222222-2222-2222-2222-222222222222', 'KLSSVIN000000002', 'Ford', 'Focus', 2020),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3', '33333333-3333-3333-3333-333333333333', 'KLSSVIN000000003', 'Honda', 'Civic', 2022)
ON CONFLICT (id) DO UPDATE
SET customer_id = EXCLUDED.customer_id,
    vin = EXCLUDED.vin,
    make = EXCLUDED.make,
    model = EXCLUDED.model,
    year = EXCLUDED.year;

INSERT INTO dealerships (id, name, address, timezone)
VALUES
    ('44444444-4444-4444-4444-444444444444', 'Downtown Keyloop Motors', '100 Main Street, Detroit, MI', 'America/Detroit')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    address = EXCLUDED.address,
    timezone = EXCLUDED.timezone;

INSERT INTO service_types (id, name, duration_minutes, required_skill_code)
VALUES
    ('55555555-5555-5555-5555-555555555551', 'Oil Change', 30, 'OIL_CHANGE'),
    ('55555555-5555-5555-5555-555555555552', 'Brake Inspection', 60, 'BRAKE_SERVICE'),
    ('55555555-5555-5555-5555-555555555553', 'Tire Replacement', 45, 'TIRE_SERVICE'),
    ('55555555-5555-5555-5555-555555555554', 'Full Service', 120, 'GENERAL_MAINTENANCE')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    duration_minutes = EXCLUDED.duration_minutes,
    required_skill_code = EXCLUDED.required_skill_code;

INSERT INTO technicians (id, dealership_id, name)
VALUES
    ('66666666-6666-6666-6666-666666666661', '44444444-4444-4444-4444-444444444444', 'Alex Morgan'),
    ('66666666-6666-6666-6666-666666666662', '44444444-4444-4444-4444-444444444444', 'Morgan Lee'),
    ('66666666-6666-6666-6666-666666666663', '44444444-4444-4444-4444-444444444444', 'Taylor Smith')
ON CONFLICT (id) DO UPDATE
SET dealership_id = EXCLUDED.dealership_id,
    name = EXCLUDED.name;

INSERT INTO technician_skills (id, technician_id, skill_code)
VALUES
    ('77777777-7777-7777-7777-777777777771', '66666666-6666-6666-6666-666666666661', 'OIL_CHANGE'),
    ('77777777-7777-7777-7777-777777777772', '66666666-6666-6666-6666-666666666661', 'GENERAL_MAINTENANCE'),
    ('77777777-7777-7777-7777-777777777773', '66666666-6666-6666-6666-666666666662', 'BRAKE_SERVICE'),
    ('77777777-7777-7777-7777-777777777774', '66666666-6666-6666-6666-666666666662', 'GENERAL_MAINTENANCE'),
    ('77777777-7777-7777-7777-777777777775', '66666666-6666-6666-6666-666666666663', 'TIRE_SERVICE'),
    ('77777777-7777-7777-7777-777777777776', '66666666-6666-6666-6666-666666666663', 'OIL_CHANGE')
ON CONFLICT (technician_id, skill_code) DO NOTHING;

INSERT INTO service_bays (id, dealership_id, name)
VALUES
    ('88888888-8888-8888-8888-888888888881', '44444444-4444-4444-4444-444444444444', 'Bay 1'),
    ('88888888-8888-8888-8888-888888888882', '44444444-4444-4444-4444-444444444444', 'Bay 2')
ON CONFLICT (id) DO UPDATE
SET dealership_id = EXCLUDED.dealership_id,
    name = EXCLUDED.name;

INSERT INTO appointments (
    id,
    customer_id,
    vehicle_id,
    dealership_id,
    service_type_id,
    technician_id,
    service_bay_id,
    start_time,
    end_time,
    status
)
VALUES (
    '99999999-9999-9999-9999-999999999991',
    '11111111-1111-1111-1111-111111111111',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
    '44444444-4444-4444-4444-444444444444',
    '55555555-5555-5555-5555-555555555551',
    '66666666-6666-6666-6666-666666666661',
    '88888888-8888-8888-8888-888888888881',
    '2026-05-04 09:00:00-04',
    '2026-05-04 09:30:00-04',
    'CONFIRMED'
)
ON CONFLICT (id) DO UPDATE
SET customer_id = EXCLUDED.customer_id,
    vehicle_id = EXCLUDED.vehicle_id,
    dealership_id = EXCLUDED.dealership_id,
    service_type_id = EXCLUDED.service_type_id,
    technician_id = EXCLUDED.technician_id,
    service_bay_id = EXCLUDED.service_bay_id,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time,
    status = EXCLUDED.status;
