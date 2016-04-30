Terminal
---

This is a mini-fork of https://github.com/golang/crypto to create a more
standalone terminal reader that leaves the output to the app.  For a simple
terminal interface, use what's built into the crypto package.  This is for
applications that need more direct control over how and where the output is
drawn to the screen (or not drawn, as the case may be).

Run the example (clone the repository and `go run example/simple.go`) to see
how this might work in an app with lots of output going on in the background.
