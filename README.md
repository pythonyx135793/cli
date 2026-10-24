# cli

Run the standard-library-only Go project from the repository root:

```sh
go test -v ./...
go vet ./...
go run .
```

The HTTP fixtures exercise the actual `workspace/pkg/github` client. They cover
sparse intermediate pages, the requested result limit, missing next-page links,
and an empty page that still advertises a next page. The issue's 15-item then
10-item example requests 30 results and returns all 25 in two HTTP requests.
