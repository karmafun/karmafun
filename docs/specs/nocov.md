In the hack directory, create a go program called `nocov.go` that takes a
coverage file name in input. For information, this file contains a firs line
containing `mode:atomic` followed by lines with the following format:

```go
github.com/karmafun/karmafun/pkg/templates/context.go:74.16,76.3 1 0
```

After the file name, the numbers are:

- StarLine.Column
- EndLine.Column
- Number of statements
- Number of times the statements were executed

The program will read th `go.mod` file in the current directory in order to get
the module name to be able to deduce the relative path of the file. Then it will
read the coverage file and for each line with 0 execution, it will process the
file looking for `//nocov` comments.

The `//nocov` comment can be on the same line as the code or on the previous
line. When the line on which the `//nocov` comment is located starts a block of
code, then all the lines of the block will be ignored until the end of the
block. A block of code is defined as a line ending with `{` and the
corresponding closing `}`.

If the line is within a block of code that is ignored, then it will be ignored
as well. It means that the corresponding line will not be output in the final
coverage file and the number of statements will be set to 0.

In order to avoid rebuilding the non covered blocks, the program will maintain a
map of the ignored blocks with the start and end line. for each file. Thus the
file AST will be parsed only once.

At the end of the processing, the program will output a new coverage file with
the same format as the input one but with the non covered lines removed.

It will also output a summary of the number of lines ignored for each file and
the total number of lines ignored.
