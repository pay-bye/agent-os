ALTER TABLE instruction_records
  DROP CONSTRAINT IF EXISTS instruction_records_result_known,
  DROP CONSTRAINT IF EXISTS instruction_records_failed_precondition_not_empty,
  DROP COLUMN IF EXISTS result,
  DROP COLUMN IF EXISTS affected_ids,
  DROP COLUMN IF EXISTS failed_precondition;
