# BulkDownload

BulkDownload is a small Go service that can either bundle requested files into a zip or generate a shell script that downloads those files directly.

## Endpoints

- `POST /zip` creates a zip job from a list of file paths
- `POST /script` creates a shell script job from a list of relative file paths
- `GET /status/{id}` returns the current job status
- `GET /download/{id}` downloads the completed zip file

## Run

```bash
go run .
```

Copy `.env.example` to `.env` if you want to override the output directory or timing settings locally.

Available config values:

- `JOBS_DIR` stores generated job artifacts such as zip files and shell scripts
- `PUBLIC_BASE_URL` is used inside generated scripts for the file download base URL and should include the scheme, for example `http://localhost:9000`
- `DOWNLOAD_ROOT_DIR` is the default local folder name used by generated scripts

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

`scripts/request.sh` returns JSON with a job id. Once the job is done, run:

```bash
bash scripts/download.sh <job-id>
```

This fetches `http://localhost:9000/scripts/<job-id>.sh` and runs it, downloading the requested files into `./mohd_data/`.

## Example

```bash
curl -X POST http://localhost:8080/zip \
  -H "Content-Type: application/json" \
  -d '{"files":["/absolute/path/to/file1.txt","/absolute/path/to/file2.txt"]}'
```

```bash
curl -X POST http://localhost:8080/script \
  -H "Content-Type: application/json" \
  -d '{"files":["rna/accession.bigwig","dna/sample.cram"]}'
```

When a script job completes, your frontend can point users at:

```bash
curl -fsSL https://download.mohd.org/scripts/<id>.sh | bash
```

The generated script downloads into `./mohd_data` by default, preserves nested paths, resumes partial downloads, and limits parallelism to 3 concurrent transfers unless the user overrides `MAX_JOBS`.
