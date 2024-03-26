package httpx

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strings"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func assertInterfaces(v interface{}) string {

	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}
	if reflect.TypeOf(v).Kind() == reflect.Slice {
		slice := reflect.ValueOf(v)
		var result strings.Builder
		for i := 0; i < slice.Len(); i++ {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(slice.Index(i).Interface().(string))
		}
		return result.String()
	}

	return ""
}

func Httpx(domain string) {
	localUtils.Logger("started httpx", 1)
	cmd := exec.Command("httpx", "-u", domain, "-title", "-x", "get", "-status-code", "-ip", "-json", "-fr")
	stdout, err := cmd.Output()
	if err != nil {
		localUtils.Logger(fmt.Sprint(err), 2)
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(stdout), &result)

	title := assertInterfaces(result["title"])
	url := assertInterfaces(result["url"])
	host := assertInterfaces(result["host"])
	scheme := assertInterfaces(result["scheme"])
	a := assertInterfaces(result["a"])
	cname := assertInterfaces(result["cname"])
	tech := assertInterfaces(result["tech"])
	status_code := assertInterfaces(result["status_code"])
	port := assertInterfaces(result["port"])
	ip := assertInterfaces(result["ip"])


	db.AddUrl(title,url,host,scheme,a,cname,tech,ip,port,status_code)
}
