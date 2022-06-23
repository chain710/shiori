package cmd

import "github.com/sirupsen/logrus"

type LogLevel logrus.Level

func (l *LogLevel) String() string {
	return logrus.Level(*l).String()
}

func (l *LogLevel) Set(s string) error {
	if lvl, err := logrus.ParseLevel(s); err != nil {
		return err
	} else {
		*l = LogLevel(lvl)
		return nil
	}
}

func (l *LogLevel) Type() string {
	return "LogLevel"
}

func (l *LogLevel) Value() logrus.Level {
	return logrus.Level(*l)
}
