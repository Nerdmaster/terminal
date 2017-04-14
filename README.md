Terminal
===

This is a highly modified mini-fork of https://github.com/golang/crypto to
create a more standalone terminal reader that gives more power to the app, adds
more recognized key sequences, and leaves the output to the app.

[See the godoc documentation](https://godoc.org/github.com/Nerdmaster/terminal)
for complete API docs.

Features
---

- Completely standalone key / line reader:
  - Unlike the Go ssh/terminal package, this is pretty simple (no inclusion of
    all those other crypto libraries)
  - Unlike many all-in-one terminal packages (like termbox), this uses
    io.Reader instead of forcing raw terminal access, so you can listen to an
    SSH socket, read from a raw terminal, or convert any arbitrary stream of
    raw bytes to keystrokes
  - Unlike just about every key-reader I found, you're not tied to a specific
    output approach; this terminal package is designed to be output-agnostic.
- Parses a wide variety of keys, tested in Windows and Linux, over ssh
  connections and local terminals
- Handles unknown sequences without user getting "stuck" (after accidentally
  hitting Alt+[, for instance)
- OnKeypress callback for handling more than just autocomplete-style situations
- AfterKeypress callback for firing off events after the built-in processing
  has already occurred

Readers
---

This package contains multiple ways to gather input from a user:

### terminal.Reader

`terminal.Reader` is very similar to the ssh terminal in Go's crypto package
except that it doesn't do any output.  It's useful for gathering input from a
user in an asynchronous way while still having niceties like a command history
and special key handling (e.g., CTRL+U deletes everything from the beginning of
the line to the cursor).  Specific needs can be addressed by wrapping this type
with another type, such as the AbsPrompt.

Internally uses KeyReader for parsing keys from the io.Reader.

Have a look at the [simple reader example](example/simple.go) to get an idea
how to use this type in an application which draws random output while
prompting the user for input and printing their keystrokes to the screen.

Note that the example has some special handling for a few keys to demonstrate
(and verify correctness of) some key interception functionality.

### terminal.Prompt

`terminal.Prompt` is the closest thing to the terminal which exists in the
crypto package.  It will draw a prompt and wait for input, handling arrow keys
and other special keys to reposition the cursor, fetch history, etc.  This
should be used in cases where the crypto terminal would normally be used, but
more complex handling is necessary, such as the on/after keypress handlers.

### terminal.AbsPrompt

`terminal.AbsPrompt` offers simple output layer on top of a terminal.Reader for
cases where a prompt should be displayed at a fixed location in the terminal.
It is tied to a given io.Writer and can be asked to draw changes to the input
since it was last drawn, or redraw itself fully, including all repositioning
ANSI codes.  Since drawing is done on command, there's no need to synchronize
writes with other output.

Internally uses KeyReader for parsing keys from the io.Reader.

Have a look at the [absprompt example](example/absprompt.go) to get an idea how
this type can simplify getting input from a user compared to building your code
on top of the simpler Reader type.

As mentioned in "features", this package isn't coupled to a particular output
approach.  Check out [the goterm example](example/goterm.go) to see how you can
use a AbsPrompt with [goterm](https://github.com/buger/goterm) - or any output
package which doesn't force its input layer on you.

### terminal.DT

`terminal.DT` is for a very simple, no-frills terminal.  It has no support for
special keys other than backspace, and is meant to just gather printable keys
from a user who may not have ANSI support.  It writes directly to the given
io.Writer with no synchronizing, as it is assumed that if you wanted complex
output to happen, you wouldn't use this.

Internally uses KeyReader for parsing keys from the io.Reader.

Have a loot at the ["dumb" example](example/dumb.go) to see how a DT can be
used for an extremely simple interface.

### terminal.KeyReader

`terminal.KeyReader` lets you read keys as they're typed, giving extremely
low-level control (for a terminal reader, anyway).  The optional `Force`
variable can be set to true if you need immediate key parsing despite the
oddities that can bring.  See the
[`ParseKey` documentation](https://godoc.org/github.com/Nerdmaster/terminal#ParseKey)
for an in-depth explanation of this.

In normal mode (`Force` is false), special keys like Escape and
Alt-left-bracket will not be properly parsed until another key is pressed due
to limitations discussed in the ParseKey documentation and the Caveats section
below.  However, users won't get "stuck", as the parser will just force-parse
sequences if more than 250ms separates one read from the next.

Take a look at the [keyreport example](example/keyreport.go) to get an idea how
to build a raw key parser using KeyReader.  You can also run it directly (`go run
example/keyreport.go`) to see what sequence of bytes a given key (or key
combination) spits out.  Note that this has special handling for Ctrl+C (exit
program) and Ctrl+F (toggle "forced" parse mode).

Caveats
---

### Terminals suck

Please note that different terminals implement different key sequences in
hilariously different ways.  What's in this package may or may not actually
handle your real-world use-case.  Terminals are simply not the right medium for
getting raw keys in any kind of consistent and guaranteed way.  As an example,
the key sequence for "Alt+F" is the same as hitting "Escape" and then "F"
immediately after.  The left arrow is the same as hitting alt+[+D.  Try it on a
command line!  In linux, at least, you can fake quite a lot of special keys
because the console is so ... weird.

### io.Reader is limited

Go doesn't provide an easy mechanism for reading from an io.Reader in a
"pollable" way.  It's already impossible to tell if alt+[ is really alt+[ or
the beginning of a longer sequence.  With no way to poll the io.Reader, this
package has to make a guess.  I tried using goroutines and channels to try to
determine when it had been long enough to force the parse, but that had its own
problems, the worst of which was you couldn't cancel a read that was just
sitting waiting.  Which meant users would have to press at least one extra key
before the app could stop listening - or else the app had to force-close an
io.ReadCloser, which isn't how you want to handle something like an ssh
connection that's meant to be persistent.

In "forced" parse mode, alt+[ will work just fine, but a left arrow can get
parsed as "alt+[" followed by "D" if the reader doesn't see the D at precisely
the same moment as the "alt+[".  But in normal mode, a user who hits alt-[ by
mistake, and tries typing numbers can find themselves "stuck" for a moment
until the reader sees that enough time has passed since their mistaken "alt+["
keystroke and the "real" keys.  Or until they hit 8 bytes' worth of keys, at
which point the key reader starts making assumptions that are likely incorrect.

Low-level reading of the keyboard would solve this problem, but this package is
meant to be as portable as possible, and able to parse input from ANYTHING
readable.  Not just a local console, but also SSH, telnet, etc.  It may even be
valuable to read keystrokes captured in a file (though I suspect that would
break things in even more hilarious ways).

### Limited testing

- Tested in Windows: cmd and PowerShell, Putty ssh into Ubuntu server
- Tested in Linux: Konsole in Ubuntu VM, tmux on Debian and Ubuntu, and a raw
  GUI-less debian VM in VMWare

Windows terminals (cmd and PowerShell) have very limited support for anything
beyond ASCII as far as I can tell.  Putty is a lot better.  If you plan to
write an application that needs to support even simple sequences like arrow
keys, you should host it on a Linux system and have users ssh in.  Shipping a
Windows binary won't work with built-in tools.

If you can test out the keyreport tool in other OSes, that would be super
helpful.

### Therefore....

If you use this package for any kind of application, just make sure you
understand the limitations.  Parsing of keys is, in many cases, done just to be
able to throw away absurd user input (like Meta+Ctrl+7) rather than end up
making wrong guesses (my Linux terminal thinks certain Meta combos should print
a list of local servers followed by the ASCII parts of the sequence).

So while you may not be able to count on specific key sequences, this package
might help you gather useful input while ignoring (many) completely absurd
sequences.
