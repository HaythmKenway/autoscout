package main

import (
	"time"
	"fmt"
) 
func main(){
	for true{
	cron()
	fmt.Println("next job in ",time.Hour/2)
	time.Sleep(time.Hour/2)
	}
	return 
}
