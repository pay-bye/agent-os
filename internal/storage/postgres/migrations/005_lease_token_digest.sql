DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM leases) THEN
    RAISE EXCEPTION 'leases must be drained before adding token digests';
  END IF;
END;
$$;

ALTER TABLE leases
  ADD COLUMN token_digest TEXT NOT NULL
    CONSTRAINT leases_token_digest_not_empty CHECK (length(btrim(token_digest)) > 0);
