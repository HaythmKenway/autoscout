package db

import(
	"github.com/HaythmKenway/autoscout/pkg/httpx"
)

func AddUrls(urls[] string) (string,error){
 res,err := httpx.Httpx(urls)
 if err != nil {
	 return "",err
 }


 return res,nil
}
