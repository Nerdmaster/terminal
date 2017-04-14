// Package terminal provides support functions for dealing with terminals, as
// commonly found on UNIX systems.
//
// This is a completely standalone key / line reader.  All types that read data
// allow anything conforming to io.Reader.  All types that write data allow
// anything conforming to io.Writer.
//
// Putting a terminal into raw mode is the most common requirement, and can be
// seen in the example.
package terminal
