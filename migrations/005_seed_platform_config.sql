SET search_path TO fresnel, public;

INSERT INTO fresnel.platform_config (key, value, updated_at)
VALUES ('default_timezone', 'UTC', now())
ON CONFLICT (key) DO NOTHING;
