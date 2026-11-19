DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM settings WHERE key IN ('knowledge_embedding', 'knowledge_reranker')) THEN
        RAISE EXCEPTION 'knowledge model settings exist; cannot restore settings constraint';
    END IF;
END $$;

ALTER TABLE settings DROP CONSTRAINT settings_key_check,
    ADD CONSTRAINT settings_key_check CHECK (key IN (
        'branding', 'authentication', 'email', 'billing', 'session_reporting', 'proxies', 'mcp', 'document_vision'
    ));
