-- Migration 000001: Rollback

DROP TRIGGER IF EXISTS update_action_records_updated_at ON action_records;
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS feedback_records CASCADE;
DROP TABLE IF EXISTS action_records CASCADE;
DROP TABLE IF EXISTS simulation_runs CASCADE;
DROP TABLE IF EXISTS decision_traces CASCADE;
DROP TABLE IF EXISTS decision_records CASCADE;
DROP TABLE IF EXISTS policies CASCADE;
DROP TABLE IF EXISTS feature_snapshots CASCADE;
DROP TABLE IF EXISTS events CASCADE;
