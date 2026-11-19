DO $$
BEGIN
    LOCK TABLE settings IN ACCESS EXCLUSIVE MODE;

    IF EXISTS (
        SELECT 1 FROM billing_transactions
        WHERE mode = 'remote' AND status NOT IN ('settled', 'released', 'rejected')
    ) OR EXISTS (
        SELECT 1 FROM wallet_billing_records WHERE status NOT IN ('confirmed', 'rejected')
    ) THEN
        RAISE EXCEPTION 'Unsettled external billing transactions must be reconciled before removing external billing';
    END IF;
END $$;

UPDATE settings
SET value = value || '{"charging_mode":"local","enabled":false}'::jsonb,
    revision = revision + 1,
    updated_at = now()
WHERE key = 'billing' AND value->>'charging_mode' = 'remote';

ALTER TABLE settings ADD CONSTRAINT settings_billing_local_mode_check
    CHECK (key <> 'billing' OR value->>'charging_mode' IS DISTINCT FROM 'remote');
