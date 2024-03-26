package notifier

import (
	"io"
	"os/exec"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func ClassifyNotification(urls []string) {
	localUtils.Logger("Notifying targets", 1)

	cmd := exec.Command("notify", "-mf", "🎯 New Target Found! \n {{data}}")
	stdin, err := cmd.StdinPipe()
	localUtils.CheckError(err)

	done := make(chan error)
	go func() {
		done <- cmd.Run()
	}()

	for _, u := range urls {
		_, err := io.WriteString(stdin, u+"\n")
		localUtils.CheckError(err)
	}

	err = stdin.Close()
	localUtils.CheckError(err)

	err = <-done
	localUtils.CheckError(err)
}
