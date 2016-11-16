package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	toml "github.com/BurntSushi/toml"
	emon "github.com/exzz/emon-api-go"
	client "github.com/influxdata/influxdb/client/v2"
)

// Command line flag
var fConfig = flag.String("f", "", "Configuration file")
var fDebug = flag.Bool("d", false, "Verbose")

// Configuration file
type collectorConfig struct {
	Emon     emonConfig
	Influxdb influxdbConfig
}

type emonConfig struct {
	URL           string
	FetchInterval int
}

type influxdbConfig struct {
	URL             string
	Username        string
	Password        string
	Database        string
	RetentionPolicy string
	Precision       string
}

var config collectorConfig

func main() {

	// Parse command line flags
	flag.Parse()
	if *fConfig == "" {
		fmt.Printf("Missing required argument -f\n")
		os.Exit(0)
	}

	if _, err := toml.DecodeFile(*fConfig, &config); err != nil {
		fmt.Printf("Cannot parse config file: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting\n")

	// set up
	emon, err := emon.NewClient(emon.Config{
		URL: config.Emon.URL,
	})
	if err != nil {
		fmt.Printf("Unable to connect to Energymonitor: %s\n", err)
		os.Exit(1)
	}

	influxdb, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.Influxdb.URL,
		Username: config.Influxdb.Username,
		Password: config.Influxdb.Password,
	})
	if err != nil {
		fmt.Print("Unable to connect to InfluxDB: %s\n", err.Error())
		os.Exit(1)
	}

	// main loop
	tickerWrite := time.NewTicker(time.Duration(config.Emon.FetchInterval) * time.Second)

	for {

		select {
		case <-tickerWrite.C:

			// read data
			err := emon.Read()
			if err != nil {
				fmt.Printf("Cannot fetch EnergyMonitor data: %s\n", err)
				continue
			}

			// create influxdb point
			bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
				Database:        config.Influxdb.Database,
				Precision:       config.Influxdb.Precision,
				RetentionPolicy: config.Influxdb.RetentionPolicy,
			})

			tags := map[string]string{"site": emon.Sensor.SiteName, "module": emon.Sensor.ModuleName}
			fields := map[string]interface{}{
				"real":     emon.Sensor.RealPower,
				"apparent": emon.Sensor.ApparentPower,
			}

			pt, err := client.NewPoint("power", tags, fields, time.Unix(int64(emon.Sensor.Time), 0))
			if err != nil {
				fmt.Printf("Cannot create infludb point: %s\n", err.Error())
				continue
			}
			bp.AddPoint(pt)

			// write infludb point
			err = influxdb.Write(bp)
			if err != nil {
				fmt.Printf("Cannot write infludb point: %s\n", err.Error())
				continue
			}

			// log read value
			if *fDebug {
				fmt.Printf("%+v\n", emon.Sensor)
			}
		}
	}

}
