DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM settings WHERE key = 'document_vision') THEN
        RAISE EXCEPTION 'document_vision setting exists; cannot restore settings constraint';
    END IF;
END $$;

ALTER TABLE settings DROP CONSTRAINT settings_key_check,
    ADD CONSTRAINT settings_key_check CHECK (key IN (
        'branding', 'authentication', 'email', 'billing', 'session_reporting', 'proxies', 'mcp'
    ));
