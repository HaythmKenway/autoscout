package httpx

import (
	"encoding/json"
	"os/exec"
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

	if slice, ok := v.([]interface{}); ok {
		var result strings.Builder
		for i, val := range slice {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(assertInterfaces(val))
		}
		return result.String()
	}

	return ""
}

func Httpx(domain string) {
	localUtils.Logger("started httpx", 1)
	cmd := exec.Command("httpx", "-u", domain, "-title", "-x", "get", "-status-code", "-ip", "-json", "-fr")
	stdout, err := cmd.Output()
	localUtils.CheckError(err)

	var result map[string]interface{}
	err = json.Unmarshal(stdout, &result)
	localUtils.CheckError(err)

	title := assertInterfaces(result["title"])
	url := assertInterfaces(result["url"])
	host := assertInterfaces(result["host"])
	scheme := assertInterfaces(result["scheme"])
	a := assertInterfaces(result["a"])
	cname := assertInterfaces(result["cname"])
	tech := assertInterfaces(result["tech"])
	statusCode := assertInterfaces(result["status_code"])
	port := assertInterfaces(result["port"])
	ip := assertInterfaces(result["ip"])

	db.AddUrl(title, url, host, scheme, a, cname, tech, ip, port, statusCode)
}
