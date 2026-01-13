# session_converter_test.go

Unit tests for gotgproto session format conversion.

## Test Cases

### TestConvertToGotgprotoSession_Success

**Scenario:** Valid session.Data → Wrapped JSON format

**Setup:**
- Input: `&session.Data{DC: 2, Addr: "149.154.167.40:443", AuthKey: [...]}`

**Expected Results:**
- No error returned
- `result.Version == 1`
- `result.Data` contains valid JSON
- Parsed JSON has structure: `{"Version":1,"Data":{"DC":2,"Addr":"...","AuthKey":"..."}}`
- `parsed["Data"]["DC"] == 2` (nested, not at root)

**Validates:**
- Proper JSON wrapping for gotgproto compatibility
- Version field set to 1
- Session data nested under "Data" key

---

### TestConvertToGotgprotoSession_NilInput

**Scenario:** Nil input → Error returned

**Expected Results:**
- Error returned: "session data is nil"
- Result is nil

## Coverage Summary

| Test | Covers |
|------|--------|
| Success | JSON wrapping, Version field, Data nesting |
| NilInput | Input validation, error handling |
