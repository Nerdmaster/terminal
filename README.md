Terminal
---

This is a highly modified mini-fork of https://github.com/golang/crypto to
create a more standalone terminal reader that gives more power to the app, adds
more recognized key sequences, and leaves the output to the app.  For a simple
terminal interface, use what's built into the crypto package.  This is for
applications that need more direct control.

Run the example (clone the repository and `go run example/simple.go`) to see
how this might work in an app with lots of output going on in the background.
