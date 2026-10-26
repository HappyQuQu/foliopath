-- +goose Up
CREATE TABLE ocr_search_documents (
 id INTEGER PRIMARY KEY,
 library_id INTEGER NOT NULL,
 fingerprint TEXT NOT NULL,
 asset_id INTEGER NOT NULL,
 text_key TEXT NOT NULL,
 UNIQUE(library_id,fingerprint,asset_id),
 FOREIGN KEY(library_id,fingerprint,asset_id) REFERENCES ocr_asset_results(library_id,fingerprint,asset_id) ON DELETE CASCADE
);
CREATE VIRTUAL TABLE ocr_search USING fts5(text_key,content='ocr_search_documents',content_rowid='id',tokenize='trigram case_sensitive 1');
-- +goose StatementBegin
CREATE TRIGGER ocr_documents_insert AFTER INSERT ON ocr_search_documents BEGIN INSERT INTO ocr_search(rowid,text_key) VALUES(NEW.id,NEW.text_key); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER ocr_documents_delete AFTER DELETE ON ocr_search_documents BEGIN INSERT INTO ocr_search(ocr_search,rowid,text_key) VALUES('delete',OLD.id,OLD.text_key); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER ocr_documents_update AFTER UPDATE OF text_key ON ocr_search_documents BEGIN INSERT INTO ocr_search(ocr_search,rowid,text_key) VALUES('delete',OLD.id,OLD.text_key); INSERT INTO ocr_search(rowid,text_key) VALUES(NEW.id,NEW.text_key); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER ocr_results_invalidate AFTER UPDATE ON ocr_asset_results BEGIN DELETE FROM ocr_search_documents WHERE library_id=OLD.library_id AND fingerprint=OLD.fingerprint AND asset_id=OLD.asset_id; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_asset_results_insert AFTER INSERT ON ocr_asset_results BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_asset_results_update AFTER UPDATE ON ocr_asset_results BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_asset_results_delete AFTER DELETE ON ocr_asset_results BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_library_settings_insert AFTER INSERT ON ocr_library_settings BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_library_settings_update AFTER UPDATE ON ocr_library_settings BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_library_settings_delete AFTER DELETE ON ocr_library_settings BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_search_documents_insert AFTER INSERT ON ocr_search_documents BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_search_documents_update AFTER UPDATE ON ocr_search_documents BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_ocr_search_documents_delete AFTER DELETE ON ocr_search_documents BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_source AFTER UPDATE OF source_fingerprint,kind,name,relative_path,directory_id,mtime_ns ON assets BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER search_revision_availability AFTER UPDATE OF status ON libraries BEGIN UPDATE catalog_search_state SET revision=revision+1 WHERE singleton_key=1; END;
-- +goose StatementEnd
-- +goose Down
DROP TRIGGER search_revision_availability;
DROP TRIGGER search_revision_source;
DROP TRIGGER search_revision_ocr_search_documents_delete;
DROP TRIGGER search_revision_ocr_search_documents_update;
DROP TRIGGER search_revision_ocr_search_documents_insert;
DROP TRIGGER search_revision_ocr_library_settings_delete;
DROP TRIGGER search_revision_ocr_library_settings_update;
DROP TRIGGER search_revision_ocr_library_settings_insert;
DROP TRIGGER search_revision_ocr_asset_results_delete;
DROP TRIGGER search_revision_ocr_asset_results_update;
DROP TRIGGER search_revision_ocr_asset_results_insert;
DROP TRIGGER ocr_results_invalidate;
DROP TRIGGER ocr_documents_update;
DROP TRIGGER ocr_documents_delete;
DROP TRIGGER ocr_documents_insert;
DROP TABLE ocr_search;
DROP TABLE ocr_search_documents;
