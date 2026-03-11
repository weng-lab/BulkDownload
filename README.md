# BulkDownload

BulkDownload is a small Go service that bundles requested files into a zip and makes that zip available for download.

## Endpoints

- `POST /zip` creates a zip job from a list of file paths
- `GET /status/{id}` returns the current job status
- `GET /download/{id}` downloads the completed zip file

## Run

```bash
go run .
```

Copy `.env.example` to `.env` if you want to override the output directory or timing settings locally.

## Example

```bash
curl -X POST http://localhost:8080/zip \
  -H "Content-Type: application/json" \
  -d '{"files":["/absolute/path/to/file1.txt","/absolute/path/to/file2.txt"]}'
```
