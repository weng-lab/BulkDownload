# BulkDownload

BulkDownload is a small Go service that can either bundle requested files into a zip or tarball, or generate a shell script that downloads those files directly.

## Endpoints

- `POST /zip` creates a zip job from a list of file paths
- `POST /tarball` creates a tarball job from a list of file paths
- `POST /script` creates a shell script job from a list of relative file paths
- `GET /status/{id}` returns the current job status
- `GET /download/{id}` downloads the completed job artifact

## Run

```bash
go run .
```

Copy `.env.example` to `.env` if you want to override the output directory or timing settings locally.

Available config values:

- `JOBS_DIR` stores generated job artifacts such as zip files and shell scripts
- `SOURCE_ROOT_DIR` is an optional base directory for `/zip` and `/tarball`; when set, clients can submit relative paths and the service resolves them under this root
- `PUBLIC_BASE_URL` is used inside generated scripts for the file download base URL and should include the scheme, for example `http://localhost:9000`
- `DOWNLOAD_ROOT_DIR` is the default local folder name used by generated scripts

If `SOURCE_ROOT_DIR` is not set, `/zip` and `/tarball` continue to accept the file paths exactly as sent by the client.

## Docker

Build the image:

```bash
docker build -t bulkdownload:latest .
```

Run it with separate mounts for generated archives and source data:

```bash
docker run -d \
  --name bulkdownload \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /srv/bulkdownload/jobs:/data/jobs \
  -v /srv/bulkdownload/source:/data/source:ro \
  -e JOBS_DIR=/data/jobs \
  -e SOURCE_ROOT_DIR=/data/source \
  -e PORT=8080 \
  bulkdownload:latest
```

Notes for container deploys:

- Mount a writable host directory into `JOBS_DIR` so completed `.zip`, `.tar.gz`, and `.sh` files survive container replacement.
- If you set `SOURCE_ROOT_DIR`, clients can submit relative paths like `study-a/sample1.bam` instead of container-specific absolute paths.
- `/zip` and `/tarball` can only read files that exist inside the container, typically via a bind mount.
- Job metadata is stored in memory only, so restarting the container forgets old job IDs even if artifact files are still present on disk.

## Local Script Test

For a quick local test, set `PUBLIC_BASE_URL=http://localhost:9000` in `.env` and restart the Go app.

From the repo root, use three terminals:

```bash
go run .
```

```bash
bash scripts/fs.sh
```

```bash
bash scripts/request.sh
```

To try archive jobs instead:

```bash
bash scripts/request-zip.sh
```

```bash
bash scripts/request-tarball.sh
```

Those default to the repo's `testdata/alpha.txt` and `testdata/bravo.txt`, but you can still pass explicit absolute paths as arguments.

Each request script returns JSON with a job id. Once a script job is done, run:

```bash
bash scripts/download.sh <job-id>
```

This fetches `${API_BASE_URL:-http://localhost:8080}/download/<job-id>` and runs it, downloading the requested files into `./mohd_data/`.

## Example

```bash
curl -X POST http://localhost:8080/zip \
  -H "Content-Type: application/json" \
  -d '{"files":["/absolute/path/to/file1.txt","/absolute/path/to/file2.txt"]}'
```

```bash
curl -X POST http://localhost:8080/tarball \
  -H "Content-Type: application/json" \
  -d '{"files":["/absolute/path/to/file1.txt","/absolute/path/to/file2.txt"]}'
```

If `SOURCE_ROOT_DIR=/data/source`, the archive endpoints can also accept relative paths:

```bash
curl -X POST http://localhost:8080/zip \
  -H "Content-Type: application/json" \
  -d '{"files":["project-a/alpha.txt","project-a/bravo.txt"]}'
```

```bash
curl -X POST http://localhost:8080/script \
  -H "Content-Type: application/json" \
  -d '{"files":["rna/accession.bigwig","dna/sample.cram"]}'
```

When a script job completes, your frontend can point users at:

```bash
curl -fsSL https://download.mohd.org/download/<id> | bash
```

The generated script downloads into `./mohd_data` by default, preserves nested paths, resumes partial downloads, and limits parallelism to 3 concurrent transfers unless the user overrides `MAX_JOBS`.
