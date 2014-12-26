package requestlog_test

import (
	"testing"

	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestFromContext(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	requestlog.FromContext(ctx).AddExecutionInfo(nil)
	assert.Equal(t, 1, requestlog.FromContext(ctx).Size())
}
