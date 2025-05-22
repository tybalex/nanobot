package log

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	debugs            = strings.Split(os.Getenv("NANOBOT_DEBUG"), ",")
	EnableMessages    = slices.Contains(debugs, "messages")
	EnableProgress    = slices.Contains(debugs, "progress")
	DebugLog          = slices.Contains(debugs, "log")
	Base64Replace     = regexp.MustCompile(`(;base64,[a-zA-Z0-9+/=]{0,12})[a-zA-Z0-9+/=]+"`)
	Base64Replacement = []byte(`$1..."`)
)

func Messages(_ context.Context, server string, out bool, data []byte) {
	if EnableProgress && bytes.Contains(data, []byte(`"notifications/progress"`)) {
	} else if EnableMessages && !bytes.Contains(data, []byte(`"notifications/progress"`)) {
	} else {
		return
	}

	fmtStr := "->(%s) %s\n"
	if !out {
		fmtStr = "<-(%s) %s\n"
	}
	data = Base64Replace.ReplaceAll(data, Base64Replacement)
	_, _ = fmt.Fprintf(os.Stderr, fmtStr, server, strings.ReplaceAll(strings.TrimSpace(string(data)), "\n", " "))
}

func Errorf(_ context.Context, format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
}

func Infof(_ context.Context, format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", args...)
}

func Debugf(_ context.Context, format string, args ...any) {
	if !DebugLog {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}
