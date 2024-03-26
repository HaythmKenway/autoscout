package notifier

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func ClassifyNotification(urls []string) {
	localUtils.Logger("Notifying targets ", 1)

	pipeReader, pipeWriter := io.Pipe()
	cmd := exec.Command("notify", "-mf", "🎯 New Target Found! \n {{data}}")
	cmd.Stdin = pipeReader
	done := make(chan error)
	go func() {
		// Start the command and capture any errors
		err := cmd.Run()
		done <- err
	}()
	for _, u := range urls {
		_, err := pipeWriter.Write([]byte(u + "\n"))
		if err != nil {
			fmt.Println(err)
		}

	}
}
