package monitor_client

import (
	"strings"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	t.Log(formatTime(time.Now()))
	end := time.Now().UnixNano()
	start := time.Now().Add(time.Duration(-1*24) * time.Hour).UnixNano()
	t.Log(start, end)
	s := "iot_service_live{ident=\"${HOST}\",instance=\"${NAME}:80\"}"
	t.Log(strings.ReplaceAll(s, "${HOST}", "server1"))
}
