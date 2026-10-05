package quiz

import "encoding/json"

// jsonUnmarshal — отдельная обёртка над encoding/json, чтобы тесты могли
// подменять только разбор JSON, не трогая движок.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
