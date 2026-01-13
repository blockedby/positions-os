# 0004_add_triggers.up.sql

Adds `updated_at` auto-update triggers for all tables.

Uses `CREATE OR REPLACE FUNCTION` + `CREATE TRIGGER` pattern.
