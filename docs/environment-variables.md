# Environment Variables Reference

This document outlines the environment variables used by the Job-Hunter OS components.

## Overview

The system consists of two primary services:

1.  **Collector & Web UI**: Handles Telegram scraping, serves the management dashboard, and provides the API.
2.  **Analyzer**: Consumes jobs from NATS and uses an LLM to extract structured data.

---

## 1. Core Infrastructure

These variables are shared by both services to connect to the database and message broker.

| Variable       | Description                   | Default                                                           |
| :------------- | :---------------------------- | :---------------------------------------------------------------- |
| `DATABASE_URL` | PostgreSQL connection string. | `postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable` |
| `NATS_URL`     | NATS connection URL.          | `nats://localhost:4222`                                           |

---

## 2. Web UI & Collector

Configures the unified web server and Telegram scraping capabilities.

| Variable            | Description                      | Default                    |
| :------------------ | :------------------------------- | :------------------------- |
| `HTTP_PORT`         | Port for the web server and API. | `3100`                     |
| `STATIC_DIR`        | Path to static assets (CSS, JS). | `./static`                 |
| `TEMPLATES_DIR`     | Path to Go HTML templates.       | `./internal/web/templates` |
| `TG_API_ID`         | Telegram API ID (numeric).       | _Required_                 |
| `TG_API_HASH`       | Telegram API Hash.               | _Required_                 |
| `TG_SESSION_STRING` | Base64 encoded Telegram session. | _Required_                 |

---

## 3. LLM / Analyzer

Configures the Analyzer's interaction with the LLM (e.g., LM Studio, Ollama, or OpenAI).

| Variable              | Description                       | Default                    |
| :-------------------- | :-------------------------------- | :------------------------- |
| `LLM_BASE_URL`        | API endpoint for the LLM.         | `http://localhost:1234/v1` |
| `LLM_MODEL`           | Model name to use for completion. | `local-model`              |
| `LLM_API_KEY`         | API Key for the LLM provider.     | (Empty)                    |
| `LLM_TEMPERATURE`     | Sampling temperature.             | `0.1`                      |
| `LLM_MAX_TOKENS`      | Maximum tokens in LLM response.   | `2048`                     |
| `LLM_TIMEOUT_SECONDS` | Timeout for LLM requests.         | `60`                       |

---

## 4. Logging & General

| Variable    | Description                                           | Default          |
| :---------- | :---------------------------------------------------- | :--------------- |
| `LOG_LEVEL` | Logging verbosity (`debug`, `info`, `warn`, `error`). | `info`           |
| `LOG_FILE`  | Path to the log file.                                 | `./logs/app.log` |

---

## Service Flow Diagrams

### Data Flow for Scraping

1. `Collector` loads `TG_API_ID`, `TG_API_HASH`, and `TG_SESSION_STRING`.
2. Connects to `DATABASE_URL` to fetch targets.
3. Scrapes messages and publishes raw jobs to NATS (`NATS_URL`).

### Data Flow for Analysis

1. `Analyzer` connects to `NATS_URL` and listens for `jobs.new`.
2. Fetches raw content from `DATABASE_URL`.
3. Sends content to LLM at `LLM_BASE_URL` using `LLM_MODEL`.
4. Saves structured results back to `DATABASE_URL`.

### Web UI Flow

1. `Web Server` starts on `HTTP_PORT`.
2. Serves files from `STATIC_DIR` and renders `TEMPLATES_DIR`.
3. Uses `DATABASE_URL` to display statistics and job lists on the dashboard.
