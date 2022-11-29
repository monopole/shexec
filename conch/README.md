# conch

`conch` is a simple shell program that
looks like it wraps a database like `mysql`.

It's intended for use in testing shell processors.

It reads commands from `stdin`,
prints stuff to `stdOut`
and `stdErr` and has no other side effects.

To see flags and commands:

```
conch help
```

The flags can be used to change the shell's behavior,
e.g. cause it to error when reading a particular database row,
or take a long time to do a query.
