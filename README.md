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
  "Registration": "open",
  "Theme": {
    "Primary_text": "000000",
    "Secondary_text": "000000",
    "Background": "ffffff",
    "Border": "000000"
  },
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

## TODO
- break up main
- refine css for chrome
- refine css for mobile platforms