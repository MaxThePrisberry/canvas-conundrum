After writing or changing any code in the server implementation, make sure that `go vet && go build && go clean -testcache && go test -timeout=30s ./...` passes without issues and `/home/nandm2/go/bin/deadcode /home/nandm2/Desktop/canvas-conundrum/server/` passes with no new entries.

When writing tests, the actual server implementation should never be changed just to make testing easier. The actual implementation should only be changed if it is found that it doesn't meet the specification given in `game-design.md` and `websocket-events.md`.
