ALTER TABLE bookmark ADD COLUMN has_archive BOOLEAN DEFAULT false;
CREATE INDEX has_archive_index on bookmark(has_archive);