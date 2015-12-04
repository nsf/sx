# Sx file format

## What for?

The main target is configuration files written and read by humans. By no means there is an intention to make it a universal textual format for data exchange. JSON is good at that role. However, it's clear that it's not well suited for manual editing. Attempts were made to fix that problem and various formats such as CSON (CoffeeScript one), YAML and TOML appeared and became somewhat popular.

## Why?

1. YAML is probably the first widely known attempt to tackle the problem, but a very big specification ruins it. It's hard to implement a YAML parser over a weekend. If you use YAML, you most likely rely on existing parser and this is a problem on its own. Universal readable configuration format should be easy to write a parser for.
2. Editing whitespace characters in YAML may lead to structural changes. That's a very bad idea. Same mistake in CSON. Using whitespace for structure works for small files and simple hierarchies, but as soon as you start editing files which occupy multiple editor screens, it starts to hurt.
3. TOML is an attempt to formally specify .ini-like format. Sadly, the part with tables, arrays and arrays of tables is not readable at all.
3. JSON is hard to edit, this is a known fact. Requires too much punctuation. Has no comments, which is important for config files. Some parsers allow trailing comma, some are not, it is not part of the specification. It's very easy to make trailing comma mistake by copy & pasting things.

Out of frustration with existing formats here comes a need to find something simple, yet beautiful. Sx is an attempt to provide that. The main inspiration comes from lisp programming languages family of course. Yes, programming in lisp can be painful, but the same idea applied to data description seems to be working.

Another big question is data types supported by a format. Some specify a few, some specify more. Sx has nothing to offer at all in this domain. All elements sx uses are lists and strings. These elements can be interpreted in a different way though, but that's outside of a format definition.

## Specification

The format is based on modern S-expression notation used widely today, for example in lisp family of programming languages. However, the format is redefined from the ground up and is not compatible with any existing formats (unless unintentionally so).

### Text encoding requirement

Format is defined in terms of ASCII byte values. Any ASCII-compatible encoding will work. The input is steam of values, not bytes, hence encodings like UTF-32 may work as well. Preferred encoding is **UTF-8**, but it's not required. Parsers may or may not support various encodings.

### Lexical elements

#### Space character

Space character is defined as one of: `\r`, `\n`, `\t`, `<space>`.

#### Non-scalar character

Non-scalar character is defined as one of: `\r`, `\n`, `\t`, `"`, `(`, `)`, `;`, `<backquote>`. It is possible to escape grave accent mark in markdown, but I don't do that and use `<backquote>` instead.

#### String literal

String literals are defined with minimal amount of escape sequences. Some escape sequences are there simply for readability purposes.

String literal starts with a quotation mark `"` and ends with a  quotation mark `"`. String literal may contain one or more: valid escape sequence or any other byte, except `\n` and `"`.

Valid escape sequences are:

- `\r` - is converted to `0x0D` byte
- `\n` - is converted to `0x0A` byte
- `\t` - is converted to `0x09` byte
- `\\` - is converted to `0x5C` byte
- `\xHH` - is converted to `0xHH` byte, `H` is a valid hex digit, upper-case or lower-case

Invalid escape sequence is an error and should not be allowed.

Example: `"\tHello, world.\x00"`

#### Raw string literals

Raw string literal starts with a `<backquote>` and ends with a `<backquote>`. You can use any byte in-between, except `\n` and `<backquote>`. There are no escape sequences. Raw strings are useful for representing regular expressions and file paths on some operating systems.

Example: `` `C:\Program Files\ABC\Data` ``

#### Multi-line string literals

Multi-line string literal is a special lexical element which contains a set of raw strings. Multi-line string literal starts with a `<backquote>` followed by `\r\n` or `\n` and ends with a `<backquote>` on a separate line, possibly preceeded by one or more spaces. In regular expression terms that would be something like: `` `$ `` and `` ^\s*` ``

Between these opening and closing lines each line can be an empty line or a line which starts with a `|` byte followed by an optional `<space>` character. In regular expression terms, multi-line string literal grabs everything between `^\s*\| ?` and `\n` (skipping `\r`) on each such non-empty line.

Example

    (welcome-message `
      | Greetings, {{name}}.
      |
      | Welcome to this wonderful place called `home`
    `)

Yields a list of two elements: "welcome-message" and

    Greetings, {{name}}.

    Welcome to this wonderful place called `home`

As you can see this scheme allows absolutely any character inside a multi-line string. You can even have a multi-line string inside a multi-line string. Nothing new in fact, inspired by single-line comment syntax in many programming languages which allows anything inside of a comment line. Another cool fact about such multi-line literals is that a pretty printer may indent them as it sees fit.

#### Scalar

Scalar starts with a first non-scalar character and ends with a last non-scalar character.

#### List

List may contain scalars, strings or other lists. List starts with an opening parenthesis `(` and ends with a closing parenthesis `)`. You can use space characters as separators for list elements, but it's not required in some cases. For example:
```
hello(iam"John")world
```
is a valid sequence of a scalar `hello`, a list with two elements `iam` (scalar) and `John` (string) followed by a scalar `world`. While this form is allowed by definition and is made intentionally for simplicity, it's not recommended. Please, use at least a single space character to separate list elements from each other. A preferred way to write the example above is:
```
hello (iam "John") world
```

#### Comment

Comment starts with a semicolon `;` and ends with a newline byte `\n`. Anything in-between is allowed.

## Reference parser

Reference parser is written in Go and tries to use none of the Go-specific features. The intention is to make it easy to port the parser into any modern programming language. However, this is only a parser alone. Integration with reflection facilities and other programmer-friendly features of languages is out of scope of a reference parser.
