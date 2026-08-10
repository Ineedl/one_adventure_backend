package httpresponse

// Response is the common HTTP response envelope used by gateway services.
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func Success(data any) Response {
	if data == nil {
		data = map[string]any{}
	}
	return Response{Code: 0, Msg: "success", Data: data}
}

func Failure(code int, msg string) Response {
	return Response{Code: code, Msg: msg, Data: map[string]any{}}
}
