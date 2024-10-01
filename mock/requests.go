package mock

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
)

func MockJSONBody(c *gin.Context, content interface{}) {
	if c.Request == nil {
		panic("request is nil")
	}
	d, err := json.Marshal(content)
	if err != nil {
		panic(err)
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(d))
}
