Write the draft alignment for poem $ARGUMENTS to /tmp/kokin-align-$ARGUMENTS.txt and open it in Helix in a new tmux window.

Execute this bash command:

```bash
cd /Users/idg/Documents/kokin-tei-merge/tools/annotate && \
GOPATH=/Users/idg/Documents/kokin-tei-merge/.go \
go run ./cmd/align-review \
  prepare \
  -hachi ../../data/hachidaishu-wordlist.xml \
  -input ../../data/kokin-annotated.xml \
  -poem  $ARGUMENTS
```

After running, tell the user: "Helix is open in tmux window align-$ARGUMENTS. Edit the draft, then run /apply-poem $ARGUMENTS when done."
