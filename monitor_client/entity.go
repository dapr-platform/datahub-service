package monitor_client

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/model"
)

type MetricsResponse struct {
	Status string      `json:"status"`
	Data   MetricsData `json:"data"`
}

type MetricsData struct {
	ResultType string          `json:"resultType"`
	Result     []MetricsResult `json:"result"`
}

type LokiQueryResultResp struct {
	Status string          `json:"status"`
	Data   LokiQueryResult `json:"data"`
}
type LokiLabelValueResp struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}
type LokiQueryResult struct {
	ResultType string       `json:"resultType"`
	Result     []LokiResult `json:"result"`
}
type LokiResult struct {
	Stream map[string]any `json:"stream"`
	Values [][]any        `json:"values"`
}

type QueryResultResp struct {
	Status string      `json:"status"`
	Data   QueryResult `json:"data"`
}

type QueryResult struct {
	Type   string      `json:"resultType"`
	Result model.Value `json:"result"`
}

func (qr *QueryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.Result = &sv
		qr.Type = "scalar"

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.Result = vv
		qr.Type = "vector"

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.Result = mv
		qr.Type = "matrix"

	default:
		qr.Type = "unknown"
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}

type MetricsResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type MetricsRangeResponse struct {
	Status string           `json:"status"`
	Data   MetricsRangeData `json:"data"`
}
type MetricsRangeData struct {
	ResultType string               `json:"resultType"`
	Result     []MetricsRangeResult `json:"result"`
}
type MetricsRangeResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}
