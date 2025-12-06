# Benchmark Results: Manual Scanner

**Environment:**
- OS: darwin
- Arch: arm64
- CPU: Apple M3

## Single Token Scanning

| Benchmark | ops | ns/op | B/op | allocs/op |
|-----------|-----|-------|------|-----------|
| Keyword | 78,417,979 | 15.25 | 0 | 0 |
| Identifier | 73,019,718 | 17.32 | 0 | 0 |
| Number | 131,816,068 | 9.12 | 0 | 0 |
| HexNumber | 100,000,000 | 10.64 | 0 | 0 |
| String | 151,201,761 | 7.93 | 0 | 0 |
| SingleChar | 192,291,577 | 7.27 | 0 | 0 |
| Comment | 167,061,290 | 7.18 | 0 | 0 |

## Full Tokenization

| Benchmark | ops | ns/op | B/op | allocs/op |
|-----------|-----|-------|------|-----------|
| Simple | 3,057,578 | 445 | 1,360 | 14 |
| Medium | 1,000,000 | 1,201 | 3,984 | 37 |
| Complex | 349,312 | 4,002 | 13,488 | 102 |
| Hex | 981,447 | 1,231 | 3,984 | 37 |
| Strings | 1,000,000 | 1,116 | 3,696 | 34 |
| ManyTokens | 346,030 | 3,432 | 13,104 | 98 |

## Helper Functions

| Benchmark | ops | ns/op | B/op | allocs/op |
|-----------|-----|-------|------|-----------|
| IsIdentChar | 145,176,751 | 8.25 | 0 | 0 |
| KeywordsLookup | 27,211,629 | 50.22 | 0 | 0 |