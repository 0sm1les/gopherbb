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
{
  "Registration": "closed",
  "Categories": [
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
}
```

## changing theme
The theme can be easily adjusted by changing the 7 color variables in the css file in the static folder, by default the website is just white and black.

## changing default username color
The default user name colors can be changed by editing the user table in gopherbb.sql, default username colors are black.

## TODO
- break up main
- refine css for chrome