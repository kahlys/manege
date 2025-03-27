CREATE OR REPLACE FUNCTION notify_config_changes()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('config_changes', TG_TABLE_NAME);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER config_change_notifier
AFTER INSERT OR UPDATE OR DELETE ON config
FOR EACH STATEMENT
EXECUTE PROCEDURE notify_config_changes();