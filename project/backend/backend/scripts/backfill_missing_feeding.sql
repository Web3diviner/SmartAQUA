-- Manual fallback for the offline-feeding backfill.
--
-- NOTE: You normally do NOT need to run this. The backend now backfills
-- these three events automatically on startup (see
-- internal/database/database.go, backfillOfflineFeedings) — push to GitHub,
-- let Render deploy, and the records appear. This script exists only as a
-- manual alternative; it is idempotent (skips rows that already exist).
--
-- Replace YOUR_DEVICE_ID with your device ID before running:
--   SELECT DISTINCT device_id FROM devices;
--
-- Times are West Africa Time (UTC+1). Actual dispensed = scheduled x Q10,
-- where Q10 factor = 2.1 ^ ((temp - 25.0) / 10)  (Clarias, firmware model).

INSERT INTO feeding_events (
    device_id, timestamp, quantity_grams, actual_dispensed, duration_seconds,
    trigger_type, result, error_message, temperature, q10_factor,
    obm_safety_factor, created_at
)
SELECT v.*, NOW()
FROM (VALUES
    -- 11 Jun 2026 5:00 PM, 104 g @ 24.9 C -> Q10 0.9926 -> 103.23 g
    ('YOUR_DEVICE_ID', TIMESTAMPTZ '2026-06-11 17:00:00+01', 104.0, 103.23, 0,
     'SCHEDULED', 0, '', 24.9, 0.9926, 1.0),
    -- 12 Jun 2026 10:56 AM, 105 g @ 25.0 C -> Q10 1.0000 -> 105.00 g
    ('YOUR_DEVICE_ID', TIMESTAMPTZ '2026-06-12 10:56:00+01', 105.0, 105.00, 0,
     'SCHEDULED', 0, '', 25.0, 1.0000, 1.0),
    -- 12 Jun 2026 5:00 PM, 104 g @ 25.3 C -> Q10 1.0225 -> 106.34 g
    ('YOUR_DEVICE_ID', TIMESTAMPTZ '2026-06-12 17:00:00+01', 104.0, 106.34, 0,
     'SCHEDULED', 0, '', 25.3, 1.0225, 1.0)
) AS v(device_id, ts, quantity_grams, actual_dispensed, duration_seconds,
       trigger_type, result, error_message, temperature, q10_factor,
       obm_safety_factor)
WHERE NOT EXISTS (
    SELECT 1 FROM feeding_events e
    WHERE e.device_id = v.device_id AND e.timestamp = v.ts
);

-- Verify:
-- SELECT id, device_id, timestamp, quantity_grams, actual_dispensed,
--        temperature, q10_factor
-- FROM feeding_events ORDER BY timestamp DESC LIMIT 5;
