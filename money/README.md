# Money

Package `money` provides exact money values for ledger entries.

Money is represented as an integer number of minor units plus an explicit currency
code. Arithmetic only succeeds for matching currencies, which keeps accidental
cross-currency addition or comparison out of the General Ledger service layer.

```go
usd, err := money.ParseDecimal("123.45", money.USD)
if err != nil {
	return err
}

fee := money.MustNewMinor(250, money.USD)
total, err := usd.Add(fee)
if err != nil {
	return err
}

fmt.Println(total.String()) // USD 125.95
```

## General Ledger Fit

The OMG General Ledger specification records two money values on an entry:

- `orig_amount`: the original transaction amount and currency.
- `amount`: the amount in the ledger reporting currency.

This package supports that model by storing the currency on every `Money` value. The
ledger service can validate that `Entry.Amount.Currency()` matches the ledger reporting
currency while allowing `Entry.OriginalAmount.Currency()` to differ.

## Currency Scale

The package ships with a small default scale registry:

- `USD`, `EUR`, and `GBP`: 2 decimal places.
- `JPY`: 0 decimal places.
- `KWD`: 3 decimal places.

Additional currencies or accounting units can be registered:

```go
err := money.RegisterCurrency("BTC", 8)
```

## JSON

Money values marshal to the API shape used by the OpenAPI contract:

```json
{"amount":"123.45","currency":"USD"}
```

The amount is a canonical decimal string. Floating-point helpers are intentionally not
part of the package because ledger arithmetic must be exact.
