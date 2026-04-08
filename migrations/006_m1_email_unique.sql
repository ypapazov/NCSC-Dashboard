SET search_path TO fresnel, public;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON fresnel.users (LOWER(email));
