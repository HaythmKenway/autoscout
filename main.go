package main

import (
	"time"
	"fmt"
	"flag"
	"github.com/HaythmKenway/autoscout/internal/db"
) 
func main(){
	tgt := flag.String("u","", "Add Host")
	deamon := flag.Bool("d",false, "Run Autoscout in deamon mode")
	msg := flag.String("m", "", "Add Message")
	flag.Parse()
	if(*deamon){
	for true{
		fmt.Println("running as deamon")
		db.Cron();
	fmt.Println("next job in ",time.Hour/2)
	time.Sleep(time.Hour/2)
	}}
	if(*msg != ""){
		fmt.Println("adding message")
		}
	if(*tgt != ""){
		fmt.Println("adding target")
		db.AddTarget(*tgt);}
	return



}
