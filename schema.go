package main

var sqlSchema = `

create table honks (honkid integer primary key, userid integer, what text, honker text, xid text, rid text, dt text, url text, audience text, noise text, convoy text, whofore integer, format text, precis text, oonker text, flags integer);
create table donks (honkid integer, fileid integer);
create table filemeta (fileid integer primary key, xid text, name text, description text, url text, media text, local integer);
create table honkers (honkerid integer primary key, userid integer, name text, xid text, flavor text, combos text, owner text);
create table xonkers (xonkerid integer primary key, name text, info text, flavor text);
create table zonkers (zonkerid integer primary key, userid integer, name text, wherefore text);
create table doovers(dooverid integer primary key, dt text, tries integer, userid integer, rcpt text, msg blob);
create table onts (ontology text, honkid integer);
create table honkmeta (honkid integer, genus text, json text);
create table hfcs (hfcsid integer primary key, userid integer, json text);

create index idx_honksxid on honks(xid);
create index idx_honksconvoy on honks(convoy);
create index idx_honkshonker on honks(honker);
create index idx_honksoonker on honks(oonker);
create index idx_honkerxid on honkers(xid);
create index idx_xonkername on xonkers(name);
create index idx_zonkersname on zonkers(name);
create index idx_filesxid on filemeta(xid);
create index idx_filesurl on filemeta(url);
create index idx_ontology on onts(ontology);
create index idx_onthonkid on onts(honkid);
create index idx_honkmetaid on honkmeta(honkid);
create index idx_hfcsuser on hfcs(userid);

create table config (key text, value text);

create table users (userid integer primary key, username text, hash text, displayname text, about text, pubkey text, seckey text, options text);
create table auth (authid integer primary key, userid integer, hash text, expiry text);
CREATE index idxusers_username on users(username);
CREATE index idxauth_userid on auth(userid);
CREATE index idxauth_hash on auth(hash);

`