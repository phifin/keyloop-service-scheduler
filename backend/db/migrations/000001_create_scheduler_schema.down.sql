DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS service_bays;
DROP TABLE IF EXISTS technician_skills;
DROP TABLE IF EXISTS technicians;
DROP TABLE IF EXISTS service_types;
DROP TABLE IF EXISTS dealerships;
DROP TABLE IF EXISTS vehicles;
DROP TABLE IF EXISTS customers;

DROP FUNCTION IF EXISTS set_updated_at();
DROP EXTENSION IF EXISTS pgcrypto;
