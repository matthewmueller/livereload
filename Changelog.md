# 0.0.7 / 2025-05-26

- remove log

# 0.0.6 / 2025-05-26

- handle external livereload with `lr.ListenAndServe(ctx, addr)`
- handle server restarts by first checking if the page is reachable before reloading. If not reachable, we'll retry with an S-curve shaped backoff algorithm.

# 0.0.5 / 2025-05-18

- trigger a reload if there's been a disconnect and then a reconnect

# 0.0.4 / 2025-04-14

- only attach script to html content that contains a </body> tag

# 0.0.3 / 2025-04-05

- **Breaking**: rename `Reload` to `Publish` and document API
- support multiple event types
- bump sse

# 0.0.2 / 2024-03-15

- add a license

# 0.0.1 / 2024-03-15

- initial commit
