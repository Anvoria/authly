DELETE FROM permissions 
WHERE service_id = '00000000-0000-0000-0000-000000000001' 
AND bit IN (4, 5, 6, 7, 8)
AND resource IS NULL;

