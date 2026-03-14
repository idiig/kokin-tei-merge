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
