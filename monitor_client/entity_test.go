package monitor_client

import (
	"encoding/json"
	"github.com/prometheus/common/model"
	"github.com/spf13/cast"
	"reflect"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	str := "{\"status\":\"success\",\"data\":{\"resultType\":\"vector\",\"result\":[{\"metric\":{\"level\":\"debug\"},\"value\":[1703747280.524,\"88129\"]},{\"metric\":{\"level\":\"error\"},\"value\":[1703747280.524,\"34172\"]},{\"metric\":{\"level\":\"info\"},\"value\":[1703747280.524,\"131\"]}],\"stats\":{\"summary\":{\"bytesProcessedPerSecond\":72920863,\"linesProcessedPerSecond\":304211,\"totalBytesProcessed\":53633803,\"totalLinesProcessed\":223750,\"execTime\":0.735506969,\"queueTime\":14.415540882,\"subqueries\":0,\"totalEntriesReturned\":3,\"splits\":48,\"shards\":768,\"totalPostFilterLines\":223750,\"totalStructuredMetadataBytesProcessed\":0},\"querier\":{\"store\":{\"totalChunksRef\":224,\"totalChunksDownloaded\":224,\"chunksDownloadTime\":33799706,\"chunk\":{\"headChunkBytes\":0,\"headChunkLines\":0,\"decompressedBytes\":47012578,\"decompressedLines\":195739,\"compressedBytes\":4789718,\"totalDuplicates\":0,\"postFilterLines\":195739,\"headChunkStructuredMetadataBytes\":0,\"decompressedStructuredMetadataBytes\":0}}},\"ingester\":{\"totalReached\":96,\"totalChunksMatched\":14,\"totalBatches\":147,\"totalLinesSent\":13807,\"store\":{\"totalChunksRef\":38,\"totalChunksDownloaded\":38,\"chunksDownloadTime\":14492511,\"chunk\":{\"headChunkBytes\":320488,\"headChunkLines\":1448,\"decompressedBytes\":6300737,\"decompressedLines\":26563,\"compressedBytes\":584523,\"totalDuplicates\":0,\"postFilterLines\":28011,\"headChunkStructuredMetadataBytes\":0,\"decompressedStructuredMetadataBytes\":0}}},\"cache\":{\"chunk\":{\"entriesFound\":224,\"entriesRequested\":224,\"entriesStored\":0,\"bytesReceived\":11451404,\"bytesSent\":0,\"requests\":213,\"downloadTime\":2007536},\"index\":{\"entriesFound\":0,\"entriesRequested\":0,\"entriesStored\":0,\"bytesReceived\":0,\"bytesSent\":0,\"requests\":0,\"downloadTime\":0},\"result\":{\"entriesFound\":0,\"entriesRequested\":0,\"entriesStored\":0,\"bytesReceived\":0,\"bytesSent\":0,\"requests\":0,\"downloadTime\":0},\"statsResult\":{\"entriesFound\":0,\"entriesRequested\":0,\"entriesStored\":0,\"bytesReceived\":0,\"bytesSent\":0,\"requests\":0,\"downloadTime\":0}}}}}"
	var queryResult QueryResultResp

	err := json.Unmarshal([]byte(str), &queryResult)
	if err != nil {
		t.Error(err)
	}
	vector := queryResult.Data.Result.(model.Vector)

	for _, sample := range vector {
		t.Log(sample.Value)
		t.Log(reflect.TypeOf(sample.Value))
		levelStr := sample.Metric["level"]
		levelVal := cast.ToInt(sample.Value.String())
		t.Log(levelStr, levelVal)
	}
	t.Log(queryResult.Data.Type)
}
