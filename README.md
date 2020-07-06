# Gnock Gnock
## Who's there? 
### Whoever you'd like.

[![Go Report](https://goreportcard.com/badge/github.com/zerbitx/gnockgnock)](https://goreportcard.com/report/github.com/zerbitx/gnockgnock)
  
Live customizable server for e2e testing

### TODO:
    1) More configurability like headers to reply with, cookies etc.
    2) Convenience methods for programmtically configuring, rather than building the structs directly.
     
    
# Goal

A server you can send a config to on the fly that interacts thusly...

```
Your test : Hey buddy if I hit this endpoint, will you send me a 409?

GnockGnock: Sure thing.

Your test: "this endpoint"

GnockGnock: 409

Your test: ✅
```

```
Your test: Hey if I hit this endpoint will you send a 200 and "You're the best!"

GnockGnock: Cool.

Your test: "this endpoint"

GnockGnock: 200: "You're the best!"

Your test: ✅
```

```
Your test: Hey will you respond to all these requests in these ways for the next 1 minute and 42 seconds

GnockGnock: K

Your test: "these endpoints in tests for 1:41"

GnockGnock: Yep, yeah, uh huh, you got it

Your test (1:43): "one of these endpoints"

GnockGnock: New config who dis?
```
