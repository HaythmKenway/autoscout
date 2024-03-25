package httpx

import (
	"fmt"
	"log"

	//	"github.com/projectdiscovery/goflags"
	"github.com/HaythmKenway/autoscout/pkg/utils"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
	"github.com/projectdiscovery/httpx/runner"
)

func Httpx(domains []string) (string, error) {
	utils.Logger("Running Httpx on targets ", 1)
	gologger.DefaultLogger.SetMaxLevel(levels.LevelVerbose)
	options := runner.Options{
		Methods:         "GET",
		InputTargetHost: domains,
		OutputIP:        true,
		StatusCode:      true,
		OnResult: func(r runner.Result) {
			// handle error
			if r.Err != nil {
				fmt.Printf("[Err] %s: %s\n", r.Input, r.Err)
				return
			}
			fmt.Printf("%s %s %d\n", r.Input, r.Host, r.StatusCode)
		},
	}

	if err := options.ValidateOptions(); err != nil {
		log.Fatal(err)
	}

	httpxRunner, err := runner.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	defer httpxRunner.Close()

	httpxRunner.RunEnumeration()
	return "", nil
}
