# DeviceCheck

A lightweight and idiomatic Go client for the Apple DeviceCheck API, supporting both production and development environments.

## Features

*   **Complete API coverage for DeviceCheck:**
    *   `update_two_bits`
    *   `query_two_bits`
    *   `validate_device_token`
*   **Customizable token provider** for your own JWT or App Attestation tokens
*   **Simple API:** just call `cli.Do(ctx, request)`
*   **Well-defined error mappings** for DeviceCheck response codes
*   **Extensive test suite** with `WantError` for expressive error assertions

## Go Version

Requires Go 1.24 or later.

## Installation

```bash
go get github.com/takimoto3/devicecheck
```

## Usage

### Initialize a client:

```go
provider := MyTokenProvider{} // implements GetToken(time.Time) (string, error)
gen := devicecheck.UUIDGenerator{} // or your custom Generator implementation
cli, err := devicecheck.NewClient(provider, gen)
if err != nil {
    log.Fatal(err)
}
```
## About appleapi options

The `devicecheck` client is built on top of the shared `appleapi-core` module:
[`https://github.com/takimoto3/appleapi-core`](https://github.com/takimoto3/appleapi-core)

The `appleapi-core` package provides a common client implementation for Apple APIs that use JWT-based authentication and signed HTTP requests.
It exposes configuration through functional options (`appleapi.Option`) that can be passed to `NewClient`. For example, to provide a custom logger:

```go
cli, err := devicecheck.NewClient(provider, gen, appleapi.WithLogger(myCustomLogger))
```

Additionally, the `devicecheck` client accepts a `devicecheck.Generator` implementation for generating `TransactionID`s. If `nil` is passed, a default `UUIDGenerator` is used.

### Use the development endpoint:

This switches the base URL from the production host to the development host.

```go
cli, err := devicecheck.NewClient(provider, gen, appleapi.WithDevelopment())
```

### Send requests:

`TransactionID` and `TimeStamp` are automatically generated if omitted. You can also explicitly provide them in the request.

#### Query bit states:

```go
req := &devicecheck.QueryRequest{
    DeviceToken:   "device-token",
}
resp, err := cli.Do(context.Background(), req)
if err == devicecheck.ErrBitStateNotFound {
    log.Println("bit state not found")
} else if err != nil {
    log.Fatalf("query failed: %v", err)
}
if resp.Result != nil {
    fmt.Println("bit0:", resp.Result.Bit0)
    fmt.Println("bit1:", resp.Result.Bit1)
    fmt.Println("last update:", resp.Result.LastUpdateTime)
}
```

To explicitly provide `TransactionID` and `TimeStamp`:

```go
reqWithDetails := &devicecheck.QueryRequest{
    DeviceToken:   "device-token",
    TransactionID: "my-custom-transaction-id",
    TimeStamp:     devicecheck.UnixTime(time.Now().UTC()),
}
resp, err = cli.Do(context.Background(), reqWithDetails)
if err == devicecheck.ErrBitStateNotFound {
    log.Println("bit state not found for custom request")
} else if err != nil {
    log.Fatalf("query with custom details failed: %v", err)
}
if resp.Result != nil {
    fmt.Println("custom bit0:", resp.Result.Bit0)
    fmt.Println("custom bit1:", resp.Result.Bit1)
    fmt.Println("custom last update:", resp.Result.LastUpdateTime)
}
```

#### Update bit states:

```go
req := &devicecheck.UpdateRequest{
    DeviceToken:   "device-token",
    Bit0:          true,
    Bit1:          false,
}
resp, err := cli.Do(context.Background(), req)
if err != nil {
    log.Fatalf("update failed: %v", err)
}
// Handle successful update, resp.Result will be nil for UpdateRequest
```

#### Validate a device token:

```go
req := &devicecheck.ValidateRequest{
    DeviceToken:   "device-token",
}
resp, err := cli.Do(context.Background(), req)
if err != nil {
    log.Fatalf("validation failed: %v", err)
}
// Handle successful validation, resp.Result will be nil for ValidateRequest
```

## Error handling

> **Note:** Apple DeviceCheck may return HTTP 200 with an error message in the body. This is not a mistake. The service sometimes encodes application-level errors in the response message while keeping the status code as 200.
> For example:
> Status: 200
> Body: "Bit State Not Found"
> 
> In this case, the client interprets it and returns the error `ErrBitStateNotFound`.

Common DeviceCheck error responses are mapped to predefined variables:

*   `ErrBadDeviceToken`
*   `ErrInvalidAuthorizationToken`
*   `ErrBitStateNotFound`
*   `ErrServerError`
*   ...

## License

This project is licensed under the [MIT License](LICENSE). 


