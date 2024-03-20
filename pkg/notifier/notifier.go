package notifier

import (
	"os/exec"
	"github.com/HaythmKenway/autoscout/pkg/utils"
)
func SendNotification(msg string,tableName string) {
	// send notification
	_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (url) VALUES (?)", tableName), msg)
		if err != nil {
			tx.Rollback()
			return err
		}

	msg = "🎯 New Target Found! \n" + msg
	cmd := exec.Command("notify", "-mf", msg) 
	cmd.Run()

}


