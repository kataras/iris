# Iris REST Types Sub-Package Documentation

## Overview

The `rest/types` sub-package provides various Go types that are ready to be used when dealing with JSON APIs and SQL drivers in Go. These types are particularly useful as field types for API request and response structures. This documentation covers the following types, and the `Parse` function of each type that has one:

- `DayTime`
- `Duration`
- `ISO8601`
- `KitchenTime`
- `Number`
- `SimpleDate`
- `Season`
- `StringOrArray`
- `TimeNotationDuration`
- `Weekday`

The helpers that are not field types (`Merge`, `GetSimpleDateRange`, `RegisterFixedLocation` and the rest) are listed at the end, under [Other exports](#other-exports).

```go
import "github.com/kataras/iris/v14/rest/types"
```

### SQL Compatibility

These types can be used to read and write to SQL columns. They implement the `sql.Scanner` and `driver.Valuer` interfaces, making them compatible with SQL databases like PostgreSQL.

## DayTime

The `DayTime` type represents a time compatible with the "hour:minute:second" format. It is useful for representing times of day without a date component, such as opening hours or event times.

### Parse Method

#### `ParseDayTime`

Parses a string into a `DayTime`.

```go
func ParseDayTime(s string) (DayTime, error)
```

##### Example

```go
dt, err := types.ParseDayTime("15:04:05")
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Time types.DayTime `json:"time"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Time types.DayTime `json:"time"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Time: types.DayTime{Hour: 15, Minute: 4, Second: 5}}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "time": "15:04:05"
}
```

## Duration

The `Duration` type is a JSON representation of the standard `time.Duration` type. It is useful for representing time intervals, such as durations of events or timeouts.

### Parse Method

#### `ParseDuration`

Parses a string into a `Duration` just like the standard `time.ParseDuration` function.

```go
func ParseDuration(s string) (Duration, error)
```

##### Example

```go
d, err := types.ParseDuration("1h30m")
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Duration types.Duration `json:"duration"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Duration types.Duration `json:"duration"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Duration: types.Duration{Duration: time.Hour + 30*time.Minute}}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "duration": "1h30m0s"
}
```

## ISO8601

The `ISO8601` type represents a time compatible with the JavaScript ISO time format. It is useful for representing timestamps in a format that is widely used in web APIs and JavaScript applications.

### Parse Method

#### `ParseISO8601`

Parses a string into an `ISO8601` time.

```go
func ParseISO8601(s string) (types.ISO8601, error)
```

##### Example

```go
iso, err := types.ParseISO8601("2024-01-02T15:04:05Z")
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Time types.ISO8601 `json:"time"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Time types.ISO8601 `json:"time"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Time: types.ISO8601{Time: time.Now()}}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "time": "2024-01-02T15:04:05Z"
}
```

## KitchenTime

The `KitchenTime` type represents a time compatible with the "3:04 PM" format. It is useful for representing times in a 12-hour clock format, commonly used in user interfaces and reports.

### Parse Method

#### `ParseKitchenTime`

Parses a string into a `KitchenTime`.

```go
func ParseKitchenTime(s string) (KitchenTime, error)
```

##### Example

```go
kt, err := types.ParseKitchenTime("3:04 PM")
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Time types.KitchenTime `json:"time"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Time types.KitchenTime `json:"time"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Time: types.KitchenTime{Hour: 15, Minute: 4, Period: "PM"}}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "time": "3:04 PM"
}
```

## Number

The `Number` type is a `float64` with a JSON null. `NaN` and the two infinities marshal as `null` and a JSON `null` unmarshals back to `NaN`; a plain `float64` fails to marshal instead, which takes the whole document with it when an average or a ratio comes out undefined. On the SQL side the same rule holds: a `NULL` column scans to `NaN` and a `NaN` value is written as `NULL`, so a nullable `float8` column needs no `sql.NullFloat64` wrapper.

The zero value is the number zero, not null. `NullNumber()` returns the null one and `IsNull()` reports it.

```go
func NullNumber() Number
func (n Number) IsNull() bool
func (n Number) Float64() float64
```

`Number` has no `Parse`: `types.Number(f)` is the conversion, and a text value goes through `Scan`.

##### Example

```go
type Response struct {
    Ratio types.Number `json:"ratio"`
}

func handleResponse(ctx iris.Context) {
    ctx.JSON(Response{Ratio: types.Number(math.NaN())})
}
```

##### Raw JSON Output

```json
{
    "ratio": null
}
```

## SimpleDate

The `SimpleDate` type represents a date in the "year-month-day" format. It is useful for representing dates without a time component, such as birthdates or event dates.

### Parse Method

#### `ParseSimpleDate`

Parses a string into a `SimpleDate`.

```go
func ParseSimpleDate(s string) (SimpleDate, error)
```

##### Example

```go
sd, err := types.ParseSimpleDate("2024-01-01")
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Date types.SimpleDate `json:"date"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Date types.SimpleDate `json:"date"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Date: types.SimpleDateFromTime(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "date": "2024-01-01"
}
```

#### URL Parameter Input Example

```go
func handleRequest(ctx iris.Context) {
    dateStr := ctx.Query().Get("date")
    date, err := types.ParseSimpleDate(dateStr)
    if err != nil {
        errors.InvalidArgument.Err(ctx, err)
        return
    }
    ctx.JSON(iris.Map{"date": date})
}
```

##### URL Example

```
GET /api/date?date=2024-10-30
```

As a struct field it needs no parsing of your own. `SimpleDate` implements `encoding.TextUnmarshaler`, so `ctx.ReadQuery`, `ctx.ReadParams`, `ctx.ReadURL`, `ctx.ReadForm` and `ctx.ReadHeader` fill a `SimpleDate` field from a `YYYY-MM-DD` value, and an absent key leaves it zero:

```go
type ReportQuery struct {
    From  types.SimpleDate `url:"from"`
    To    types.SimpleDate `url:"to"`
    Limit int              `url:"limit"`
}

func listReports(ctx iris.Context) {
    var q ReportQuery
    if err := ctx.ReadQuery(&q); err != nil {
        errors.InvalidArgument.Err(ctx, err)
        return
    }
    ctx.JSON(q)
}
```

```
GET /api/reports?from=2024-09-01&to=2024-09-30&limit=3
```

A `time.Time` field in the same readers expects RFC 3339 (`2024-09-01T00:00:00Z`). The matching `MarshalText` is what the YAML, XML and MsgPack encoders write for a `SimpleDate`, the same `2024-09-01` text. JSON keeps going through `MarshalJSON` and `UnmarshalJSON`.

## Season

The `Season` type is a bitmask of seasons (Winter, Spring, Summer, Autumn). It is useful for representing multiple seasons in a single value, such as seasonal availability or preferences.

### Handler Example

#### Input

```go
type Request struct {
    Season types.Season `json:"season"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Season types.Season `json:"season"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Season: types.Winter}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "season": 1
}
```

#### Multiple Seasons Example

```go
type Request struct {
    Seasons types.Season `json:"seasons"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Seasons types.Season `json:"seasons"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Seasons: types.Winter | types.Spring}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "seasons": 3
}
```

#### String Method

The `String` method returns the string representation of the season(s).

##### Example

```go
season := Winter | Spring
fmt.Println(season.String()) // Output: "Winter, Spring"
```

## StringOrArray

The `StringOrArray` type can be a **single** string or an **array** of strings. It marshals and unmarshals as a single string joined by commas. This is useful for handling flexible input formats in APIs.

### UnmarshalJSON Method

#### `UnmarshalJSON`

Parses JSON data into a `StringOrArray`.

```go
func (soa *StringOrArray) UnmarshalJSON(data []byte) error
```

##### Example

```go
var soa types.StringOrArray
err := json.Unmarshal([]byte(`["value1", "value2"]`), &soa)
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Values types.StringOrArray `json:"values"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Values types.StringOrArray `json:"values"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Values: types.StringOrArray(`["value1","value2"]`)}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "values": "value1,value2"
}
```

## TimeNotationDuration

The `TimeNotationDuration` type represents a duration in a human-readable format, such as "hours:minutes:seconds". It is useful for representing durations or fixed times in a format that is easy for users to understand.

### Parse Method

#### `ParseTimeNotationDuration`

Parses a string into a `TimeNotationDuration`.

```go
func ParseTimeNotationDuration(s string) (TimeNotationDuration, error)
```

##### Example

```go
tnd, err := types.ParseTimeNotationDuration("01:30:00")
if err != nil {
    log.Fatal(err)
}
```

### Handler Example

#### Input

```go
type Request struct {
    Duration types.TimeNotationDuration `json:"duration"`
}

func handleRequest(ctx iris.Context) {
    req, ok := ctx.BindJSON[Request]()
    if !ok {
        return // 400 already written.
    }
    ctx.OK(req, nil)
}
```

#### Output

```go
type Response struct {
    Duration types.TimeNotationDuration `json:"duration"`
}

func handleResponse(ctx iris.Context) {
    resp := Response{Duration: types.TimeNotationDuration(time.Hour + 30*time.Minute)}
    ctx.JSON(resp)
}
```

##### Raw JSON Output

```json
{
    "duration": "01:30:00"
}
```

## Weekday

The `Weekday` type is a `time.Weekday` which travels as a lowercase name, so a client reading a schedule gets a word it can print instead of a number whose base it has to guess. The numbering is `time.Weekday`'s, Sunday being zero, which means the zero value marshals as `"sunday"`.

### Parse Method

#### `ParseWeekday`

Parses a string into a `Weekday`, in any case and in the long or the three letter form: `"Monday"`, `"monday"`, `"MON"` and `"mon"` are the same day.

```go
func ParseWeekday(s string) (Weekday, error)
```

##### Example

```go
d, err := types.ParseWeekday("MON")
if err != nil {
    log.Fatal(err)
}

d.String() // "monday"
d.Std()    // time.Monday
```

The seven days are exported as constants (`types.Sunday` .. `types.Saturday`), `Std()` converts out to `time.Weekday` and `WeekdayOf(time.Weekday)` converts in.

#### Output

```go
type Response struct {
    Day types.Weekday `json:"day"`
}

func handleResponse(ctx iris.Context) {
    ctx.JSON(Response{Day: types.Weekday(time.Monday)})
}
```

##### Raw JSON Output

```json
{
    "day": "monday"
}
```

The SQL column is text: `Value` writes the lowercase name and `Scan` reads a text column by name or an integer one by `time.Weekday`'s numbering.

## Other exports

The package also carries a few helpers that are not field types.

- `Merge(values ...any) json.Marshaler` and `MergeJSON(values ...any) ([]byte, error)` combine several values into one JSON object. Each non-nil value must marshal to an object; a `string` or `[]byte` is taken as raw JSON; nil values and `{}` contribute nothing. Overlapping keys keep the first position and the last value. The parts marshal with the framework's json/v2 policy, so a struct renders the same inside `Merge` as through `ctx.JSON`. `Merged` is the type `Merge` returns; its `Values` field holds the parts.

```go
ctx.JSON(types.Merge(user, iris.Map{"role": "admin"}))
```

- `SimpleDates` is a `[]SimpleDate` with `Contains(date)` and `DateStrings()`. `GetSimpleDateRange(date, typ, startWeekday, endWeekday)` returns the dates of the week or the month around `date`: `typ` is a `DateRangeType` (`DayRange`, `WeekRange`, `MonthRange`, `YearRange`), but only `WeekRange` and `MonthRange` are implemented; the other two panic. The two weekdays are used by `WeekRange` only.

```go
day := types.SimpleDateFromTime(time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC))
week := types.GetSimpleDateRange(day, types.WeekRange, time.Monday, time.Sunday)
week.Contains(day.AddDate(0, 0, 4)) // true: 2024-01-14 is the Sunday of that week
```

- `RegisterFixedLocation(name, secondsFromUTC)` teaches the time parsers a zone abbreviation, for input such as `2023-02-04T09:48:14+03:00` that should print as `EEST`. Call it once at program start: `types.RegisterFixedLocation("EEST", 3*60*60)`.

- `Exampler` is the interface (`ListExamples() any`) the package's tests use to generate sample values for each type. Implement it on your own types only if you reuse that test helper.
