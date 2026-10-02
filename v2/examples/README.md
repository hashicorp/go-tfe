# go-tfe v2 examples

Each directory contains a standalone example that can be run from this directory.
Set `TFE_TOKEN` to an API token. `TFE_ADDRESS` is optional and defaults to
`https://app.terraform.io`.

```sh
export TFE_TOKEN=example
export TFE_ADDRESS=https://app.eu.terraform.io
```

## Account details

Get details for the authenticated user:

```sh
go run ./account-details
```

## Account password

Change the authenticated user's password:

```sh
go run ./account-password \
  -old-password='current password' \
  -new-password='new password'
```

## Organizations

List organizations, including subscription data:

```sh
go run ./organizations-list
```

## Response headers

Get account details and print the response headers:

```sh
go run ./inspect-response-headers
```
