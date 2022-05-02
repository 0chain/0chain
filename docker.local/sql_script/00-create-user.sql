CREATE extension ltree;
CREATE DATABASE events_db;
\connect events_db;
CREATE USER zchain_user WITH ENCRYPTED PASSWORD 'zchian';
GRANT ALL PRIVILEGES ON DATABASE events_db TO zchain_user;