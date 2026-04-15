-- Production seed: Bulgarian NIS2 essential sectors (1-10) with sub-sectors,
-- plus a cross-sector "Национални органи" grouping for national CERTs/authorities.
-- Idempotent: ON CONFLICT DO NOTHING throughout.

SET search_path TO fresnel, public;

-- Top-level sectors
INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth)
VALUES
    ('b0000000-0000-4000-8000-000000000001'::uuid, NULL, 'Енергетика', '/energy/', 1),
    ('b0000000-0000-4000-8000-000000000002'::uuid, NULL, 'Транспорт', '/transport/', 1),
    ('b0000000-0000-4000-8000-000000000003'::uuid, NULL, 'Банков сектор', '/banking/', 1),
    ('b0000000-0000-4000-8000-000000000004'::uuid, NULL, 'Инфраструктури на финансовия пазар', '/finmarket/', 1),
    ('b0000000-0000-4000-8000-000000000005'::uuid, NULL, 'Здравеопазване', '/health/', 1),
    ('b0000000-0000-4000-8000-000000000006'::uuid, NULL, 'Питейна вода', '/water/', 1),
    ('b0000000-0000-4000-8000-000000000007'::uuid, NULL, 'Отпадъчни води', '/wastewater/', 1),
    ('b0000000-0000-4000-8000-000000000008'::uuid, NULL, 'Цифрова инфраструктура', '/digital/', 1),
    ('b0000000-0000-4000-8000-000000000009'::uuid, NULL, 'Управление на услуги в областта на ИКТ', '/ict/', 1),
    ('b0000000-0000-4000-8000-00000000000a'::uuid, NULL, 'Космическо пространство', '/space/', 1),
    ('b0000000-0000-4000-8000-00000000000b'::uuid, NULL, 'Национални органи', '/authorities/', 1)
ON CONFLICT (id) DO NOTHING;

-- Sub-sectors: Енергетика
INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth)
VALUES
    ('b0000000-0000-4000-8000-000000000101'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'Електроенергия', '/energy/electricity/', 2),
    ('b0000000-0000-4000-8000-000000000102'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'Районно отопление и охлаждане', '/energy/heating/', 2),
    ('b0000000-0000-4000-8000-000000000103'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'Нефт', '/energy/oil/', 2),
    ('b0000000-0000-4000-8000-000000000104'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'Природен газ', '/energy/gas/', 2),
    ('b0000000-0000-4000-8000-000000000105'::uuid, 'b0000000-0000-4000-8000-000000000001'::uuid, 'Водород', '/energy/hydrogen/', 2)
ON CONFLICT (id) DO NOTHING;

-- Sub-sectors: Транспорт
INSERT INTO fresnel.sectors (id, parent_sector_id, name, ancestry_path, depth)
VALUES
    ('b0000000-0000-4000-8000-000000000201'::uuid, 'b0000000-0000-4000-8000-000000000002'::uuid, 'Въздушен', '/transport/air/', 2),
    ('b0000000-0000-4000-8000-000000000202'::uuid, 'b0000000-0000-4000-8000-000000000002'::uuid, 'Железопътен', '/transport/rail/', 2),
    ('b0000000-0000-4000-8000-000000000203'::uuid, 'b0000000-0000-4000-8000-000000000002'::uuid, 'Воден', '/transport/water/', 2),
    ('b0000000-0000-4000-8000-000000000204'::uuid, 'b0000000-0000-4000-8000-000000000002'::uuid, 'Автомобилен', '/transport/road/', 2)
ON CONFLICT (id) DO NOTHING;

-- National authority organizations
INSERT INTO fresnel.organizations (id, sector_id, name)
VALUES
    ('b0000000-0000-4000-8000-000000000020'::uuid, 'b0000000-0000-4000-8000-00000000000b'::uuid, 'CERT.bg'),
    ('b0000000-0000-4000-8000-000000000021'::uuid, 'b0000000-0000-4000-8000-00000000000b'::uuid, 'ДАНС'),
    ('b0000000-0000-4000-8000-000000000022'::uuid, 'b0000000-0000-4000-8000-00000000000b'::uuid, 'ГДБОП'),
    ('b0000000-0000-4000-8000-000000000023'::uuid, 'b0000000-0000-4000-8000-00000000000b'::uuid, 'МО')
ON CONFLICT (id) DO NOTHING;
