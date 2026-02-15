CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS operator_profiles (
  user_id UUID PRIMARY KEY,
  region VARCHAR(64),
  role VARCHAR(32) NOT NULL DEFAULT 'operator',
  display_name VARCHAR(255),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operator_profiles_region ON operator_profiles(region);
CREATE INDEX IF NOT EXISTS idx_operator_profiles_role ON operator_profiles(role);
