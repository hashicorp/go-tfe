package tfe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tfeAPI struct {
	ID                string                   `jsonapi:"primary,tfe"`
	Name              string                   `jsonapi:"attr,name"`
	CreatedAt         time.Time                `jsonapi:"attr,created-at,iso8601"`
	Enalbed           bool                     `jsonapi:"attr,enalbed"`
	Emails            []string                 `jsonapi:"attr,emails"`
	Status            tfeAPIStatus             `jsonapi:"attr,status"`
	StatusTimestamps  tfeAPITimestamps         `jsonapi:"attr,status-timestamps"`
	DeliveryResponses []tfeAPIDeliveryResponse `jsonapi:"attr,delivery-responses"`
}

type tfeAPIDeliveryResponse struct {
	Body string `jsonapi:"attr,body"`
	Code int    `jsonapi:"attr,code"`
}

type tfeAPIStatus string

type tfeAPITimestamps struct {
	QueuedAt time.Time `jsonapi:"attr,queued-at,rfc3339"`
}

const (
	tfeAPIStatusNormal tfeAPIStatus = "normal"
)

func Test_unmarshalResponse(t *testing.T) {
	t.Run("unmarshal properly formatted json", func(t *testing.T) {
		// This structure is intended to include multiple possible fields and
		// formats that are valid for JSON:API
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "tfe",
				"id":   "1",
				"attributes": map[string]interface{}{
					"name":       "terraform",
					"created-at": "2016-08-17T08:27:12Z",
					"enabled":    "true",
					"status":     tfeAPIStatusNormal,
					"emails":     []string{"test@hashicorp.com"},
					"delivery-responses": []interface{}{
						map[string]interface{}{
							"body": "<html>",
							"code": 200,
						},
						map[string]interface{}{
							"body": "<body>",
							"code": 300,
						},
					},
					"status-timestamps": map[string]string{
						"queued-at": "2020-03-16T23:15:59+00:00",
					},
				},
			},
		}
		byteData, errMarshal := json.Marshal(data)
		require.NoError(t, errMarshal)
		responseBody := bytes.NewReader(byteData)

		unmarshalledRequestBody := tfeAPI{}
		err := unmarshalResponse(responseBody, &unmarshalledRequestBody)
		require.NoError(t, err)
		queuedParsedTime, err := time.Parse(time.RFC3339, "2020-03-16T23:15:59+00:00")
		require.NoError(t, err)

		assert.Equal(t, unmarshalledRequestBody.ID, "1")
		assert.Equal(t, unmarshalledRequestBody.Name, "terraform")
		assert.Equal(t, unmarshalledRequestBody.Status, tfeAPIStatusNormal)
		assert.Equal(t, len(unmarshalledRequestBody.Emails), 1)
		assert.Equal(t, unmarshalledRequestBody.Emails[0], "test@hashicorp.com")
		assert.Equal(t, unmarshalledRequestBody.StatusTimestamps.QueuedAt, queuedParsedTime)
		assert.NotEmpty(t, unmarshalledRequestBody.DeliveryResponses)
		assert.Equal(t, len(unmarshalledRequestBody.DeliveryResponses), 2)
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[0].Body, "<html>")
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[0].Code, 200)
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[1].Body, "<body>")
		assert.Equal(t, unmarshalledRequestBody.DeliveryResponses[1].Code, 300)
	})

	t.Run("can only unmarshal Items that are slices", func(t *testing.T) {
		responseBody := bytes.NewReader([]byte(""))
		malformattedItemStruct := struct {
			*Pagination
			Items int
		}{
			Items: 1,
		}
		err := unmarshalResponse(responseBody, &malformattedItemStruct)
		require.Error(t, err)
		assert.Equal(t, err, ErrItemsMustBeSlice)
	})

	t.Run("can only unmarshal a struct", func(t *testing.T) {
		payload := "random"
		responseBody := bytes.NewReader([]byte(payload))

		notStruct := "not a struct"
		err := unmarshalResponse(responseBody, notStruct)
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("%v must be a struct or an io.Writer", notStruct))
	})
}

func Test_EncodeQueryParams(t *testing.T) {
	t.Run("with no listOptions and therefore no include field defined", func(t *testing.T) {
		urlVals := map[string][]string{
			"include": {},
		}
		requestURLquery := encodeQueryParams(urlVals)
		assert.Equal(t, requestURLquery, "")
	})
	t.Run("with listOptions setting multiple include options", func(t *testing.T) {
		urlVals := map[string][]string{
			"include": {"workspace", "cost_estimate"},
		}
		requestURLquery := encodeQueryParams(urlVals)
		assert.Equal(t, requestURLquery, "include=workspace%2Ccost_estimate")
	})
}

func Test_RegistryBasePath(t *testing.T) {
	client, err := NewClient(&Config{
		Token: "foo",
	})
	require.NoError(t, err)

	t.Run("ensures client creates a request with registry base path", func(t *testing.T) {
		path := "/api/registry/some/path/to/resource"
		req, err := client.NewRequest("GET", path, nil)
		require.NoError(t, err)

		expected := os.Getenv("TFE_ADDRESS") + path
		assert.Equal(t, req.retryableRequest.URL.String(), expected)
	})
}
