# validation_test.go

Unit tests for request validation — table-driven tests for `ScrapeRequest.Validate()` and `UntilTime()`.

## Test Cases: Validate()

| Test | Input | Expected |
|------|-------|----------|
| empty_request | `{}` | `ErrChannelRequired` |
| valid_channel | `channel: "@golang_jobs"` | nil |
| valid_channel_without_at | `channel: "golang_jobs"` | nil (normalizes to without @) |
| valid_with_limit | `channel: "@test", limit: 100` | nil |
| negative_limit | `channel: "@test", limit: -1` | `ErrInvalidLimit` |
| valid_date_format | `channel: "@test", until: "2024-01-15"` | nil |
| invalid_date_format | `channel: "@test", until: "not-a-date"` | `ErrInvalidDate` |
| invalid_date_wrong_order | `channel: "@test", until: "15-01-2024"` | `ErrInvalidDate` |
| future_date | `channel: "@test", until: "2099-12-31"` | `ErrFutureDate` |
| topic_ids_without_forum | `channel: "@test", topic_ids: [1,15,28]` | nil (validated at runtime) |

## Test Cases: UntilTime()

| Test | Input | Expected |
|------|-------|----------|
| empty_until | `until: ""` | nil |
| valid_date | `until: "2024-06-15"` | `*time.Time` with Year=2024 |

## Coverage Summary

**Validated Rules:**
- Either `target_id` OR `channel` must be provided
- `@` prefix stripped from channel
- `limit` must be ≥ 0
- `until` must be `YYYY-MM-DD` format
- `until` cannot be in the future
- `topic_ids` passed through (forum validation happens at runtime)

**Not Validated Here:**
- Whether channel actually exists (requires network call)
- Whether `topic_ids` are valid for the forum (checked in `service.go`)
