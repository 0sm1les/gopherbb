# gopherbb
gopherbb is a simple, easy to use forum framework written in Golang. It utilizes htmx on the frontend to maintain simplicity and postgres as its dbms.

## environment variables
```
gopherbb_gin_log
gopherbb_main_log
gopherbb_conf
gopherbb_cookie_key
gopherbb_postgres_addr
gopherbb_postgres_creds
gopherbb_postgres_db
gopherbb_salt
gopherbb_console_log
```

## config example
```
[
 {
  "Category": "general",
  "Sections": [
   {
    "Section": "discussion",
    "Id": "discussion"
   },
   {
    "Section": "reviews",
    "Id": "reviews"
   },
   {
    "Section": "tutorials",
    "Id": "tutorials"
   }
  ]
 },
 {
  "Category": "programming",
  "Sections": [
   {
    "Section": "web dev",
    "Id": "web-dev"
   },
   {
    "Section": "network programming",
    "Id": "network-programming"
   },
   {
    "Section": "system programming",
    "Id": "system-programming"
   }
  ]
 }
]
```

## changing theme
The theme can be easily adjusted by changing the 7 color variables in the css file in the static folder, by default the website is just white and black.

## changing default username color
The default user name colors can be changed by editing the user table in gopherbb.sql, default username colors are black.

## TODO
- add security options in config files: account creation cool down, post cool down.
- add option for mods and admins to pin posts
- add option for mods and admins to remove posts
- add option for mods and admins to ban users
- add option in config that limits where user roles can post
- add option in config that specifies forum visibility
- add option in config to make a forum invite only
- implement invites
- refine css for chrome