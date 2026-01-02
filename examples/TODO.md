# go-tfe SDK Examples

This repository is a wrapper around a code-generated API client. The examples serve as documentation
for typical client usage, and loosely reflect the tag taxonomy of the API specification.

I'd like you to generate example client code using the operations as defined in
#FILE:openapi/spec.json

### Requirements

- Each tag name should take the form of a separate package under #FILE:examples/, while each operation
with that tag should take the form of a separate command, as defined in
#FILE:examples/cmd/tfe/main.go

- If an operation has required parameters or request body attributes, the command should support named
arguments.

### Testing

There will be no unit tests within the examples. Test for output matching the specification using
the example command itself, along with the developer-speicific environment variables. The command
should output raw JSON:

`TFE_ADDRESS=https://tfcdev-5cc4bd34.ngrok.app TFE_TOKEN=bcroft go run examples/cmd/tfe/main.go account details | jq`
