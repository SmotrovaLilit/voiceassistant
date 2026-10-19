# Voice Assistant

Upload a recording, transcribe it with Whisper, extract workflows with Qwen, and save tasks and visit reports in [Google
Sheets CRM](https://docs.google.com/spreadsheets/d/18w6rG088znD_1-Pj-xrZ1XAf_qSV_Y33BuT9FKDLOCo/edit?usp=sharing). The
API processes requests synchronously and does not store recordings or
results locally.

## Setup

Install Go 1.26.1 or later, Docker with Docker Compose, and Make.

## Run

From the repository root:

```sh
make run
```

## Test Report

| Audio Text                                                                                                                     | Audio File                                      | Run request                                                 | API response screenshot                                                    | Google Sheets screenshot                                                                    |
|--------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|-------------------------------------------------------------|----------------------------------------------------------------------------|---------------------------------------------------------------------------------------------|
| I met Anna from Agile Robots today. They are interested in one hundred licenses. Send a proposal and follow up next Friday.    | [Anna](audioexamples/anna-agile-robots.wav)     | `make request`                                              | [HTTP 201: two tasks and one visit report](demo/api-response.jpg)          | [Tasks](demo/google-sheets-tasks.jpg) · [Visit report](demo/google-sheets-visitreports.jpg) |
| I visited David from Siemens today. Their engineering team is evaluating fifty licenses. The budget has already been approved. | [David](audioexamples/david-siemens-visit.wav)  | `make request AUDIO=audioexamples/david-siemens-visit.wav`  | [HTTP 201: one visit report](demo/siemens-api-response.jpg)                | [Visit report](demo/siemens-google-sheets.jpg)                                              |
| Remind me next Tuesday to call Emma from Bosch about renewing twenty five licenses.                                            | [Emma](audioexamples/emma-bosch-reminder.wav)   | `make request AUDIO=audioexamples/emma-bosch-reminder.wav`  | [HTTP 201: one task](demo/bosch-api-response.jpg)                          | [Task](demo/bosch-google-sheets.jpg)                                                        |
| Half a second of noise, no speech.                                                                                             | [Noise](audioexamples/unintelligible-noise.wav) | `make request AUDIO=audioexamples/unintelligible-noise.wav` | [HTTP 503: processing_unavailable](demo/unintelligible-audio-response.jpg) | No new CRM records.                                                                         |
