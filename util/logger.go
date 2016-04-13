package util

import "github.com/Sirupsen/logrus"

// ContextLogger is a wrapper for logrus that defines which package and
// function a log is written
type ContextLogger struct {
	*logrus.Entry
}

// NewContextLogger returns a new ContextLogger for the given package
func NewContextLogger(pkg string) ContextLogger {
	contextLogger := ContextLogger{logrus.WithField("package", pkg)}
	return contextLogger
}

// InFunc is a helper method to set the func field for the logger
func (c ContextLogger) InFunc(function string) ContextLogger {
	c.Entry = c.WithField("func", function)
	return c
}

func (c ContextLogger) InStruct(s string) ContextLogger {
	c.Entry = c.WithField("struct", s)
	return c
}
