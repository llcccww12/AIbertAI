ALTER TABLE experts
    ADD COLUMN dsh_plugins jsonb CHECK (jsonb_typeof(dsh_plugins) = 'array'),
    ADD COLUMN canvas_tools jsonb CHECK (jsonb_typeof(canvas_tools) = 'array');
