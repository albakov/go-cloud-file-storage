-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users_sessions
(
    id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL,
    refresh_token VARCHAR(255)    NOT NULL UNIQUE,
    expires_at    DATETIME        NOT NULL,
    CONSTRAINT `users_sessions_user_id_fn`
        FOREIGN KEY (user_id) REFERENCES users (id)
            ON DELETE CASCADE
            ON UPDATE NO ACTION
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users_sessions;
-- +goose StatementEnd
