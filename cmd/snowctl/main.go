package main

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/snowflakedb/gosnowflake"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/cmd"
)

func init() {
	logrus.SetOutput(io.Discard)
	logger := gosnowflake.GetLogger()
	logger.SetOutput(io.Discard)
	_ = logger.SetLogLevel("OFF")
}

func main() {
	cmd.Execute()
}
