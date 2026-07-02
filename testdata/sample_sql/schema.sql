-- Sample SQL schema

CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    amount REAL
);

CREATE VIEW active_users AS
    SELECT * FROM users WHERE id > 0;

CREATE PROCEDURE clear_orders()
BEGIN
    DELETE FROM orders;
END;
