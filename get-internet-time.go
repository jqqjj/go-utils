package utils

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

func GetInternetTime() (time.Time, error) {
	var (
		err      error
		now      time.Time
		resp     *http.Response
		respData []byte
		respTime struct {
			SysTime2 string `json:"sysTime2"`
			SysTime1 string `json:"sysTime1"`
		}
	)

	if resp, err = http.Get("https://quan.suning.com/getSysTime.do"); err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if respData, err = io.ReadAll(resp.Body); err != nil {
		return time.Time{}, err
	}
	if err = json.Unmarshal(respData, &respTime); err != nil {
		return time.Time{}, err
	}
	if now, err = time.ParseInLocation("2006-01-02 15:04:05", respTime.SysTime2, time.Local); err != nil {
		return time.Time{}, err
	}
	return now, nil
}
