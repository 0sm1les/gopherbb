CREATE DATABASE gopher_bb;

CREATE TABLE users (
    id SERIAL PRIMARY KEY NOT NULL,
    role varchar(12) CHECK (role in ('unranked', 'ranked', 'mod', 'admin')) DEFAULT 'unranked' NOT NULL,
    profile_pic varchar(255) DEFAULT 'default.png' NOT NULL,
    username varchar(16) NOT NULL,
    password varchar(65) NOT NULL,
    bio varchar(255) DEFAULT '' NOT NULL,
    user_fg_color varchar(6) DEFAULT '000000' NOT NULL,
    user_bg_color varchar(6) DEFAULT '000000' NOT NULL,
    custom_primary_1 varchar(6) DEFAULT '000000' NOT NULL,
    custom_primary_2 varchar(6) DEFAULT '000000' NOT NULL,
    custom_background_1 varchar(6) DEFAULT 'ffffff' NOT NULL,
    custom_background_2 varchar(6) DEFAULT 'ffffff' NOT NULL,
    date_joined timestamp without time zone NOT NULL
);

CREATE TABLE notifications (
    id SERIAL PRIMARY KEY NOT NULL,
    to_uid int references users(id) NOT NULL,
    from_uid int references users(id) NOT NULL,
    read boolean DEFAULT false NOT NULL,
    msg varchar(255) NOT NULL
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY NOT NULL,
    poster int references users(id) NOT NULL,
    section varchar(32) NOT NULL,
    status varchar(8) CHECK (status in ('draft', 'posted', 'deleted')) NOT NULL,
    title varchar(64) NOT NULL,
    md TEXT NOT NULL,
    html TEXT NOT NULL,
    time_posted timestamp without time zone NOT NULL,
    ts tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(md, '')), 'B')
) STORED
);

CREATE TABLE likes (
    id SERIAL PRIMARY KEY NOT NULL,
    post int references posts(id) NOT NULL,
    liked_by int references users(id) NOT NULL,
    time_liked timestamp without time zone NOT NULL
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY NOT NULL,
    poster int references users(id) NOT NULL,
    parent_post int references posts(id) NOT NULL,
    parent_comment int,
    status varchar(8) CHECK (status in ('posted', 'deleted')) DEFAULT 'posted' NOT NULL,
    md TEXT NOT NULL,
    html TEXT NOT NULL,
    time_posted timestamp without time zone NOT NULL
);


CREATE USER gopherbb_user WITH ENCRYPTED PASSWORD '<INSERT PASSWORD HERE>';

GRANT SELECT, INSERT, UPDATE on users TO gopherbb_user;
GRANT SELECT, INSERT, UPDATE, DELETE on likes TO gopherbb_user;
GRANT SELECT, INSERT, UPDATE on notifications TO gopherbb_user;
GRANT SELECT, INSERT, UPDATE on posts TO gopherbb_user;
GRANT SELECT, INSERT, UPDATE on comments TO gopherbb_user;

GRANT USAGE, SELECT,UPDATE on users_id_seq TO gopherbb_user;
GRANT USAGE, SELECT,UPDATE on likes_id_seq TO gopherbb_user;
GRANT USAGE, SELECT,UPDATE on notifications_id_seq TO gopherbb_user;
GRANT USAGE, SELECT,UPDATE on posts_id_seq TO gopherbb_user;
GRANT USAGE, SELECT,UPDATE on comments_id_seq TO gopherbb_user;
