// +build !windows

package main

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"

	"github.com/lxc/lxd/shared/logger"
)

func (c *execCmd) getStdout() io.WriteCloser {
	return os.Stdout
}

func (c *execCmd) getTERM() (string, bool) {
	return os.LookupEnv("TERM")
}

func (c *execCmd) controlSocketHandler(control *websocket.Conn) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGWINCH)

	for {
		sig := <-ch

		logger.Debugf("Received '%s signal', updating window geometry.", sig)

		err := c.sendTermSize(control)
		if err != nil {
			logger.Debugf("error setting term size %s", err)
			break
		}
	}

	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	control.WriteMessage(websocket.CloseMessage, closeMsg)
}
