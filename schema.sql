
CREATE TABLE honks (honkid integer primary key, userid integer, what text, honker text, xid text, rid text, dt text, url text, audience text, noise text);
CREATE TABLE donks (honkid integer, fileid integer);
CREATE TABLE files(fileid integer primary key, xid text, name text, url text, media text, content blob);
CREATE TABLE honkers (honkerid integer primary key, userid integer, name text, xid text, flavor text, pubkey text);

create index idx_honksxid on honks(xid);
create index idx_honkshonker on honks(honker);
create index idx_honkerxid on honkers(xid);
create index idx_filesxid on files(xid);
create index idx_filesurl on files(url);

CREATE TABLE config (key text, value text);

CREATE TABLE users (userid integer primary key, username text, hash text, displayname text, about text, pubkey text, seckey text);
CREATE TABLE auth (authid integer primary key, userid integer, hash text);
CREATE INDEX idxusers_username on users(username);
CREATE INDEX idxauth_userid on auth(userid);
CREATE INDEX idxauth_hash on auth(hash);

