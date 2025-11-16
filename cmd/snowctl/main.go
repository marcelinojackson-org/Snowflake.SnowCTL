package main

import (
	"io"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/cmd"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(io.Discard)
}

func main() {
	cmd.Execute()
}
