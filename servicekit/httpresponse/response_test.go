package httpresponse

import (
	"encoding/json"
	"testing"
)

func TestResponseFactories(t *testing.T) {
	success := Success(map[string]any{"token": "access"})
	if success.Code != 0 || success.Msg != "" || success.Data == nil {
		t.Fatalf("Success() = %#v", success)
	}
	failure := Failure(400, "bad request")
	if failure.Code != 400 || failure.Msg != "bad request" || failure.Data == nil {
		t.Fatalf("Failure() = %#v", failure)
	}
	encoded, err := json.Marshal(failure)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(encoded) != `{"code":400,"msg":"bad request","data":{}}` {
		t.Fatalf("encoded failure = %s", encoded)
	}
}
