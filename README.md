Terminal
---

This is a highly modified mini-fork of https://github.com/golang/crypto to
create a more standalone terminal reader that gives more power to the app, adds
more recognized key sequences, and leaves the output to the app.  For a simple
terminal interface, use what's built into the crypto package.  This is for
applications that need more direct control or handling of weird key sequences.

Features
===

- Completely standalone key / line reader (i.e., doesn't depend on the rest of
  the crypto code)
- Parses a wide variety of keys, at least on linux terminals
- Attaches to any io.Reader interface, much like the original implementation
- Delays between partial sequence bytes will result in an attempt to parse the
  previous parts of the sequence; e.g., Esc + [ + D won't be treated as a left
  arrow unless all three keys are within 250ms of each other.
  - Since this detection can't happen until the extra keys are pressed, an app
    still won't know that "Escape" was pressed until after the user types in
    another key.
- terminal.KeyReader can be used to process raw keys as they're typed instead
  of just reading lines at a time.  Optional "forced" mode can be enabled to
  read special keys as-is instead of waiting for key sequences to complete
  (e.g., reading Escape or Alt+[)
- terminal.Reader reads lines from the terminal with optional OnKeypress
  callback and a customizable maximum line length.  The Reader can be queried
  for current line and cursor position, but it doesn't draw anything to the
  screen itself.
- terminal.Prompter wraps a Reader to allow for a statically positioned prompt.
  It requires an io.Writer and can be called upon to write any changes to the
  writer since the last write.  This is suitable for an input line in a
  command-line application that needs to gather input from a static location on
  the screen while drawing other items to the screen.

Examples
===

Take a look at the [keyreport example](example/keyreport.go) to get an idea how
to build a raw key parser.  You can also run it directly (`go run
example/keyreport.go`) to see what sequence of bytes a given key (or key
combination) spits out.  Note that this has special handling for Ctrl+C (exit
program) and Ctrl+F (toggle "forced" parse mode).

You can also look at the [simple reader example](example/simple.go) or the
[prompter example](example/prompter.go) to get an idea how to use the simple
Reader and Prompter types, and how to tie it all together to build a
line-reading console application that can handle background ANSI insanity while
reading keys.  Both applications can be exited via Ctrl+D on a blank line.
Note that there is some very unusual KeyEvent magic happening in the simple
reader in order to verify some functionality that's not as easy to test
automatically.

Caveats
===

#### Terminals suck

Please note that different terminals implement different key sequences in
hilariously different ways.  What's in this package may or may not actually
handle your real-world use-case.  Terminals are simply not the right medium for
getting raw keys in any kind of consistent and guaranteed way.  As an example,
the key sequence for "Alt+F" is the same as hitting "Escape" and then "F"
immediately after.  The left arrow is the same as hitting alt+[+D.  Try it on a
command line!  In linux, at least, you can fake quite a lot of special keys
because the console is so ... weird.

#### io.Reader is limited

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
parsed as "alt+[" followed by "D" if there's a break between the two.  But in
normal mode, a user who hits alt-[ by mistake, and tries typing numbers can
find themselves "stuck" until they hit enough keys to overflow the maximum
(currently 8 bytes), which allows the reader to start throwing away bytes to
avoid breaking.

Low-level reading of the keyboard would solve this problem, but this package is
meant to be able to parse input from ANYTHING readable.  Not just a local
console, but also SSH, telnet, etc.  It may even be valuable to read keystrokes
captured in a file (though I suspect that would break things in even more
hilarious ways).

#### Limited testing


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

#### Therefore....

If you use this package for any kind of application, just make sure you
understand the limitations.  Parsing of keys is, in many cases, done just to be
able to throw away absurd user input (like Meta+Ctrl+7) rather than end up
making wrong guesses (my Linux terminal thinks certain Meta combos should print
a list of local servers followed by the ASCII parts of the sequence).

So while you may not be able to count on specific key sequences, this package
might help you gather useful input while ignoring (many) completely absurd
sequences.
