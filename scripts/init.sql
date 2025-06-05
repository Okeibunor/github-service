-- Create the database if it doesn't exist
CREATE DATABASE github_service;

-- Connect to the database
\c github_service;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create the schema
CREATE SCHEMA IF NOT EXISTS public;

-- Set default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO postgres;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO postgres;

-- Set timezone to UTC
ALTER DATABASE github_service SET timezone TO 'UTC'; 