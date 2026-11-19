BEGIN;

CREATE OR REPLACE FUNCTION session_turns_mirror() RETURNS trigger
LANGUAGE plpgsql SET search_path FROM CURRENT AS $$
DECLARE progress session_turns_partition_state; marker text := coalesce(current_setting('monkeyai.turns_mirror',true),'');
    destination regclass; assignments text; conflict_columns text; data jsonb; sid uuid; verified boolean;
BEGIN
    IF marker<>'' THEN RETURN NULL; END IF;
    SELECT * INTO STRICT progress FROM session_turns_partition_state;
    IF TG_ARGV[0]='reverse' AND progress.target_relation<>to_regclass('session_turns') THEN RETURN NULL; END IF;
    IF TG_ARGV[0]='forward' AND progress.source_relation<>to_regclass('session_turns') THEN
        RAISE EXCEPTION '轮次月分区已切换，请重试旧事务' USING ERRCODE='40001';
    END IF;
    data := CASE WHEN TG_OP='DELETE' THEN to_jsonb(OLD) ELSE to_jsonb(NEW) END;
    sid := (data->>'session_id')::uuid;
    -- Lock even unverified caches, so calibration cannot race an uncommitted
    -- INSERT/DELETE whose counter update would otherwise skip the parent row.
    -- Counters never change the session key. Keeping the same lock strength
    -- as the reporting transaction allows foreign-key checks from settlement
    -- without creating a lock upgrade on the parent row.
    SELECT turn_count_verified INTO verified FROM sessions WHERE id=sid FOR NO KEY UPDATE;
    IF TG_OP<>'UPDATE' AND verified THEN
        UPDATE sessions SET turn_count=turn_count+CASE WHEN TG_OP='INSERT' THEN 1 ELSE -1 END WHERE id=sid;
    END IF;
    IF TG_ARGV[0]='forward' OR progress.mirror_legacy THEN
        destination := CASE WHEN TG_ARGV[0]='forward' THEN progress.target_relation ELSE progress.source_relation END;
        conflict_columns := CASE WHEN TG_ARGV[0]='forward' THEN 'session_id,turn_index,started_at' ELSE 'session_id,turn_index' END;
        PERFORM set_config('monkeyai.turns_mirror','mirror',true);
        IF TG_OP='DELETE' THEN
            EXECUTE format('DELETE FROM %s WHERE session_id=$1 AND turn_index=$2 AND started_at=$3',destination)
                USING OLD.session_id,OLD.turn_index,OLD.started_at;
        ELSE
            SELECT string_agg(format('%1$I=EXCLUDED.%1$I',attname),',' ORDER BY attnum) INTO assignments
                FROM pg_attribute WHERE attrelid=destination AND attnum>0 AND NOT attisdropped AND attname NOT IN ('session_id','turn_index','started_at');
            EXECUTE format('INSERT INTO %s SELECT (jsonb_populate_record(NULL::%s,$1)).* ON CONFLICT (%s) DO UPDATE SET %s',destination,destination,conflict_columns,assignments)
                USING data;
        END IF;
        PERFORM set_config('monkeyai.turns_mirror',marker,true);
    END IF;
    IF TG_OP='DELETE' THEN
        DELETE FROM session_turns_keys WHERE session_id=OLD.session_id AND turn_index=OLD.turn_index;
    END IF;
    RETURN NULL;
END;
$$;

COMMIT;
