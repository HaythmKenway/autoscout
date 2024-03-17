package main

import (
	"time"
	"fmt"
	"flag"
) 
func main(){
	for true{
	cron()
	tgt := flag.String("tgt","", "target url")
	flag.Parse()
	if(*tgt != ""){
		addTarget(*tgt);}
	fmt.Println("next job in ",time.Hour/2)
	time.Sleep(time.Hour/2)
	}
	return 
}
