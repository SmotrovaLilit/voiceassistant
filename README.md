# Voice Assistant

Upload a recording, transcribe it with Whisper, extract workflows with Qwen, and save tasks and visit reports in Google Sheets CRM. The API processes requests synchronously and does not store recordings or results locally.

## Setup

Install Go 1.26.1 or later, Docker with Docker Compose, and Make.

Create the local API configuration once:

```sh
cp applications/voiceassistant-api/.env.example applications/voiceassistant-api/.env
```

Set `GOOGLE_SPREADSHEET_ID` and `GOOGLE_CREDENTIALS_FILE` in that `.env` file. Use an absolute path to a Google service account JSON key. Enable the Google Sheets API for its Google Cloud project and share the spreadsheet with the service account's `client_email` as an Editor. The `.env` file is ignored by Git; keep the credentials file outside the repository.

The API creates `CRM Tasks` and `CRM Sales Visit Reports` tabs if missing. Existing tabs can contain additional columns; the API maps its values by column name.

## Run

From the repository root:

```sh
make run
```

This starts LocalAI, waits for it to become healthy, loads the API configuration from `.env`, and runs the API at `http://127.0.0.1:8081`. Keep this terminal open. The first startup downloads the models and backends and may take several minutes. Subsequent starts reuse the Docker volumes.

To start only LocalAI, use `make up`. To run only the API, use `make -C applications/voiceassistant-api run`.

## Try the recording

Open [request.http](request.http) in GoLand or IntelliJ IDEA and click **Run** for a speech recording or the noise-only error example. In VS Code, use the REST Client extension and click **Send Request**. The audio files are already included in the requests.

The Anna recording says:

> I met Anna from Agile Robots today.
> They are interested in one hundred licenses.
> Send a proposal and follow up next Friday.

The expected response is HTTP `201` with the transcription and three workflows:

- A visit report about Anna from Agile Robots and the interest in 100 licenses.
- A task to send Anna a proposal.
- A task to call Anna, with a reminder on the first Friday strictly after the upload date in `Europe/Berlin`. Only a date is needed; no time was requested.

The records appear in Google Sheets before the API returns success. Each upload creates new CRM records. Model-generated wording may vary. The real CPU run took about one minute; the request allows up to ten minutes. If you change the API port, update the URL in `request.http`.

