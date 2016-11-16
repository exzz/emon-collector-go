# emon-collector-go
Read arduino open energy monitor and log data to influxdb

## Quickstart

- Clone repo
- Download gdm dependencies manager ```go get github.com/sparrc/gdm```
- Download dependencies ```gdm restore```
- Edit sample.conf
- Run ```go run emon-collector.go -f sample.conf -d```
