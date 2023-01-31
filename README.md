# Text centric fork of honk

While it uses even way less resources and is faster than vanilla honk (no
backend needed for starters), I recommend to read the following changes before.
It reports as being "muffled" for a reason ;)

Despite being a firm fork, the database code and layout are and will be
interoperable with vanilla honk. Upstream honk fixes/improvements are
backported when it must (bug fixes, security, activitypub compliance), or is
desirable (new features).

## Frontend

- Drop avatar display
- Don't display images for everyone, just put links to them
- Peaceful dark/light blue theme with decent contrast, normal font size and
  color for all the honks. Supports light and dark modes.
- "Semi material" UI, with CSS flex where needed for mobile and reduced DOM
  maximum depth (only 7!). As a result it's skinnier, tidier and draws faster
  than vanilla honk.
- Menu has been simplified, with many links to stuff i don't use removed and is
  inline with the rest of the page, and their backend functions removed (see
  there has well)
- scroll down automatically to the "oldest newest" honk after a refresh
- xzone has been removed, instead you can insert "https://..." links to honk or
  honkers in the search field if you want to import them (like you would do in
  mastodon)
- Has background image for guests sessions if you put an image named
  `background.jpg` in `data\views`.

## Backend

- Nodeinfo support so the rest of fediverse knows that honk exists :)
- Progressive web app support so it looks like a native app on smartphones
- Nuke hoot feature (twitter integration)
- Don't save external content, link it with description instead. It's actually
  not a totally bad idea on a single user instance; you only download
  attachments that you want to see from other instances.
- honk's backend has been totally removed, you can't upload attachments, use
  emus, banners, custom avatars etc. As such, funzone has been nuked as well,
  and data/blob.db is non existent. The only way to add medias in this fork is
  to upload them somewhere in your webserver and do a link to them.
- Drop filtering, let's live a dangerous life or block instance at firewall
  level from orbit
- Drop import from twitter and mastodon (but it is easy to bring it back, see
  https://github.com/mascaldotfr/honk/commit/8c58bf254e556482d7e2930f45308044958387fd)
- Drop chatter, RSS, places, events and flag features. Combos, xzone and
  ontologies have been removed but you have still the search feature as cheap
  replacements.
- Remote quoting is not implemented (see https://github.com/mascaldotfr/honk/commit/5de338e8fcd7732e3a5d13ee0d968d43d346f1b2 for reason)
- API support is present, but the following actions are not available:
  donks, most zonkit (save/unsave, ack/deack, react, zonvoy)
- Drop the 250 honks limit in the "home" timeline, display all honks received
  in the last 24 hours instead. It may sound dangerous, but due to not having
  to deal with attachments and simplified DOM tree, it is not.


## Honk's original README

=======================================================================

honk

-- features

Take control of your honks and join the federation.
An ActivityPub server with minimal setup and support costs.
Spend more time using the software and less time operating it.

No attention mining.
No likes, no faves, no polls, no stars, no claps, no counts.

Purple color scheme. Custom emus. Memes too.
Avatars automatically assigned by the NSA.

The button to submit a new honk says "it's gonna be honked".

The honk mission is to work well if it's what you want.
This does not imply the goal is to be what you want.

-- build

It should be sufficient to type make after unpacking a release.
You'll need a go compiler version 1.16 or later. And libsqlite3.

Even on a fast machine, building from source can take several seconds.

Development sources: hg clone https://humungus.tedunangst.com/r/honk

-- setup

honk expects to be fronted by a TLS terminating reverse proxy.

First, create the database. This will ask four questions.
./honk init
username: (the username you want)
password: (the password you want)
listenaddr: (tcp or unix: 127.0.0.1:31337, /var/www/honk.sock, etc.)
servername: (public DNS name: honk.example.com)

Then run honk.
./honk

-- upgrade

old-honk backup `date +backup-%F`
./honk upgrade
./honk

-- documentation

There is a more complete incomplete manual. This is just the README.

-- guidelines

One honk per day, or call it an "eighth-tenth" honk.
If your honk frequency changes, so will the number of honks.

The honk should be short, but not so short that you cannot identify it.

The honk is an animal sign of respect and should be accompanied by a
friendly greeting or a nod.

The honk should be done from a seat and in a safe area.

It is considered rude to make noise in a place of business.

The honk may be made on public property only when the person doing
the honk has the permission of the owner of that property.

-- disclaimer

Do not use honk to contact emergency services.
