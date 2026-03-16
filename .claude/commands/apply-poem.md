Read /tmp/kokin-align-$ARGUMENTS.txt and apply the alignment to the annotated XML for poem $ARGUMENTS.

Execute this bash command:

```bash
cd /Users/idg/Documents/kokin-tei-merge/tools/annotate && \
GOPATH=/Users/idg/Documents/kokin-tei-merge/.go \
go run ./cmd/align-review \
  apply \
  -hachi  ../../data/hachidaishu-wordlist.xml \
  -input  ../../data/kokin-annotated.xml \
  -output ../../data/kokin-annotated.xml \
  -poem   $ARGUMENTS
```

Report the result to the user. If validation fails, Helix will reopen automatically — tell the user to fix the highlighted error and run /apply-poem $ARGUMENTS again.

If successful, ask the user: "Commit this alignment to git? (y/n)"

Only if the user confirms, follow the git workflow: stage `data/kokin-annotated.xml`, then commit with message `feat(align): annotate poem $ARGUMENTS`.

After the commit, ask the user: "Align the next poem ($ARGUMENTS+1)? (y/n)"

Only if the user confirms, execute /align-poem with $ARGUMENTS+1.
